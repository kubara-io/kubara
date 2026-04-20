package tui

import (
	"errors"
	"fmt"
	"kubara/assets/envmap"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
)

var ErrUserCancelled = errors.New("interactive initialization cancelled")

type Answers struct {
	ProjectName                 string
	ProjectStage                string
	DockerconfigBase64          string
	ArgocdWizardAccountPassword string
	ArgocdGitHttpsURL           string
	ArgocdGitPatOrPassword      string
	ArgocdGitUsername           string
	DomainName                  string
	ArgocdHelmRepoURL           string
	ArgocdHelmRepoUsername      string
	ArgocdHelmRepoPassword      string
	Services                    map[string]bool
}

type field struct {
	Label      string
	Value      string
	Required   bool
	Secret     bool
	Hint       string
	TargetName string
}

type serviceOption struct {
	Key     string
	Label   string
	Enabled bool
}

type mode int

const (
	modeQuestions mode = iota
	modeServices
	modeReview
	modeDone
)

type model struct {
	mode             mode
	fields           []field
	fieldIndex       int
	input            textinput.Model
	progressBar      progress.Model
	serviceOptions   []serviceOption
	serviceCursor    int
	reviewConfirmed  bool
	cancelled        bool
	errMsg           string
}

func AnswersFromEnvMap(e *envmap.EnvMap) Answers {
	return Answers{
		ProjectName:                 e.ProjectName,
		ProjectStage:                e.ProjectStage,
		DockerconfigBase64:          e.DockerconfigBase64,
		ArgocdWizardAccountPassword: e.ArgocdWizardAccountPassword,
		ArgocdGitHttpsURL:           e.ArgocdGitHttpsUrl,
		ArgocdGitPatOrPassword:      e.ArgocdGitPatOrPassword,
		ArgocdGitUsername:           e.ArgocdGitUsername,
		DomainName:                  e.DomainName,
		ArgocdHelmRepoURL:           e.ArgocdHelmRepoUrl,
		ArgocdHelmRepoUsername:      e.ArgocdHelmRepoUsername,
		ArgocdHelmRepoPassword:      e.ArgocdHelmRepoPassword,
		Services: map[string]bool{
			"argocd":                  false,
			"cert-manager":            true,
			"external-dns":            true,
			"external-secrets":        true,
			"kube-prometheus-stack":   true,
			"traefik":                 true,
			"kyverno":                 true,
			"kyverno-policies":        true,
			"kyverno-policy-reporter": true,
			"loki":                    true,
			"homer-dashboard":         true,
			"oauth2-proxy":            true,
			"metrics-server":          false,
			"metallb":                 false,
			"longhorn":                false,
		},
	}
}

func (a Answers) ApplyToEnvMap(e *envmap.EnvMap) {
	e.ProjectName = strings.TrimSpace(a.ProjectName)
	e.ProjectStage = strings.TrimSpace(a.ProjectStage)
	e.DockerconfigBase64 = strings.TrimSpace(a.DockerconfigBase64)
	e.ArgocdWizardAccountPassword = strings.TrimSpace(a.ArgocdWizardAccountPassword)
	e.ArgocdGitHttpsUrl = strings.TrimSpace(a.ArgocdGitHttpsURL)
	e.ArgocdGitPatOrPassword = strings.TrimSpace(a.ArgocdGitPatOrPassword)
	e.ArgocdGitUsername = strings.TrimSpace(a.ArgocdGitUsername)
	e.DomainName = strings.TrimSpace(a.DomainName)
	e.ArgocdHelmRepoUrl = strings.TrimSpace(a.ArgocdHelmRepoURL)
	e.ArgocdHelmRepoUsername = strings.TrimSpace(a.ArgocdHelmRepoUsername)
	e.ArgocdHelmRepoPassword = strings.TrimSpace(a.ArgocdHelmRepoPassword)
}

func RunInitialConfigWizard(prefill Answers) (Answers, error) {
	m := newModel(prefill)
	program := tea.NewProgram(m)
	result, err := program.Run()
	if err != nil {
		return Answers{}, err
	}
	finalModel, ok := result.(model)
	if !ok {
		return Answers{}, fmt.Errorf("unexpected model type: %T", result)
	}
	if finalModel.cancelled {
		return Answers{}, ErrUserCancelled
	}
	if !finalModel.reviewConfirmed {
		return Answers{}, ErrUserCancelled
	}
	return finalModel.toAnswers(), nil
}

func newModel(prefill Answers) model {
	input := textinput.New()
	input.Focus()
	input.Width = 72
	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.Width = 32

	fields := []field{
		{Label: "Project Name", Value: prefill.ProjectName, Required: true, Hint: "Required", TargetName: "PROJECT_NAME"},
		{Label: "Project Stage", Value: prefill.ProjectStage, Required: true, Hint: "Required", TargetName: "PROJECT_STAGE"},
		{Label: "Docker Config Base64", Value: prefill.DockerconfigBase64, Required: true, Hint: "Required", TargetName: "DOCKERCONFIG_BASE64"},
		{Label: "Argo CD Wizard Password", Value: prefill.ArgocdWizardAccountPassword, Required: true, Secret: true, Hint: "Required", TargetName: "ARGOCD_WIZARD_ACCOUNT_PASSWORD"},
		{Label: "Argo CD Git HTTPS URL", Value: prefill.ArgocdGitHttpsURL, Required: true, Hint: "Required", TargetName: "ARGOCD_GIT_HTTPS_URL"},
		{Label: "Argo CD Git PAT/Password", Value: prefill.ArgocdGitPatOrPassword, Required: true, Secret: true, Hint: "Required", TargetName: "ARGOCD_GIT_PAT_OR_PASSWORD"},
		{Label: "Argo CD Git Username", Value: prefill.ArgocdGitUsername, Required: true, Hint: "Required", TargetName: "ARGOCD_GIT_USERNAME"},
		{Label: "Domain Name", Value: prefill.DomainName, Required: true, Hint: "Required", TargetName: "DOMAIN_NAME"},
		{Label: "Argo CD Helm Repo URL", Value: prefill.ArgocdHelmRepoURL, Required: false, Hint: "Optional", TargetName: "ARGOCD_HELM_REPO_URL"},
		{Label: "Argo CD Helm Repo Username", Value: prefill.ArgocdHelmRepoUsername, Required: false, Hint: "Optional", TargetName: "ARGOCD_HELM_REPO_USERNAME"},
		{Label: "Argo CD Helm Repo Password", Value: prefill.ArgocdHelmRepoPassword, Required: false, Secret: true, Hint: "Optional", TargetName: "ARGOCD_HELM_REPO_PASSWORD"},
	}

	serviceOrder := []string{
		"argocd",
		"cert-manager",
		"external-dns",
		"external-secrets",
		"kube-prometheus-stack",
		"traefik",
		"kyverno",
		"kyverno-policies",
		"kyverno-policy-reporter",
		"loki",
		"homer-dashboard",
		"oauth2-proxy",
		"metrics-server",
		"metallb",
		"longhorn",
	}
	serviceOptions := make([]serviceOption, 0, len(serviceOrder))
	for _, key := range serviceOrder {
		serviceOptions = append(serviceOptions, serviceOption{
			Key:     key,
			Label:   key,
			Enabled: prefill.Services[key],
		})
	}

	m := model{
		mode:           modeQuestions,
		fields:         fields,
		fieldIndex:     0,
		input:          input,
		progressBar:    progressBar,
		serviceOptions: serviceOptions,
	}
	m.configureInputForCurrentField()
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		barWidth := typed.Width - 10
		if barWidth < 10 {
			barWidth = 10
		}
		m.progressBar.Width = barWidth
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.mode {
		case modeQuestions:
			return m.updateQuestionMode(typed)
		case modeServices:
			return m.updateServicesMode(typed)
		case modeReview:
			return m.updateReviewMode(typed)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateQuestionMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.errMsg = ""
	switch msg.String() {
	case "enter":
		current := &m.fields[m.fieldIndex]
		current.Value = m.input.Value()
		if current.Required && strings.TrimSpace(current.Value) == "" {
			m.errMsg = fmt.Sprintf("%s is required", current.TargetName)
			return m, nil
		}
		if current.Required && strings.TrimSpace(current.Value) == "<...>" {
			m.errMsg = fmt.Sprintf("%s must not be <...>", current.TargetName)
			return m, nil
		}

		if m.fieldIndex == len(m.fields)-1 {
			m.mode = modeServices
			return m, nil
		}
		m.fieldIndex++
		m.configureInputForCurrentField()
		return m, nil
	case "shift+tab", "up":
		if m.fieldIndex > 0 {
			m.fields[m.fieldIndex].Value = m.input.Value()
			m.fieldIndex--
			m.configureInputForCurrentField()
		}
		return m, nil
	case "tab", "down":
		m.fields[m.fieldIndex].Value = m.input.Value()
		if m.fieldIndex < len(m.fields)-1 {
			m.fieldIndex++
			m.configureInputForCurrentField()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateServicesMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.serviceCursor > 0 {
			m.serviceCursor--
		}
	case "down", "j":
		if m.serviceCursor < len(m.serviceOptions)-1 {
			m.serviceCursor++
		}
	case " ":
		m.serviceOptions[m.serviceCursor].Enabled = !m.serviceOptions[m.serviceCursor].Enabled
	case "backspace":
		m.mode = modeQuestions
		m.fieldIndex = len(m.fields) - 1
		m.configureInputForCurrentField()
	case "enter":
		m.mode = modeReview
	}
	return m, nil
}

func (m model) updateReviewMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y", "enter":
		m.reviewConfirmed = true
		m.mode = modeDone
		return m, tea.Quit
	case "n", "backspace":
		m.mode = modeQuestions
		m.fieldIndex = 0
		m.configureInputForCurrentField()
	}
	return m, nil
}

func (m *model) configureInputForCurrentField() {
	current := m.fields[m.fieldIndex]
	m.input.SetValue(current.Value)
	m.input.CursorEnd()
	m.input.Placeholder = current.Hint
	m.input.CharLimit = 8192
	if current.Secret {
		m.input.EchoMode = textinput.EchoPassword
		m.input.EchoCharacter = '*'
	} else {
		m.input.EchoMode = textinput.EchoNormal
	}
}

func (m model) View() string {
	header := "kubara init wizard\n\n"
	footer := "\nctrl+c/esc: cancel"

	switch m.mode {
	case modeQuestions:
		current := m.fields[m.fieldIndex]
		progress := m.progressBar.ViewAs(m.questionProgress()) + fmt.Sprintf(" %d/%d", m.fieldIndex+1, len(m.fields))
		errLine := ""
		if m.errMsg != "" {
			errLine = "\nError: " + m.errMsg + "\n"
		}
		return header + "\n" + current.Label + " (" + current.Hint + ")\n" + m.input.View() + "\n" + errLine + "\n" + progress + "\n\nenter: next | tab/down: skip | shift+tab/up: previous" + footer
	case modeServices:
		var b strings.Builder
		b.WriteString(header)
		b.WriteString("Service Selection\n")
		b.WriteString("space: toggle | enter: continue | ↑/↓: move\n\n")
		for i, opt := range m.serviceOptions {
			cursor := " "
			if i == m.serviceCursor {
				cursor = ">"
			}
			check := " "
			if opt.Enabled {
				check = "x"
			}
			b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, check, opt.Label))
		}
		b.WriteString(footer)
		return b.String()
	case modeReview:
		return header + m.reviewView() + "\n\nConfirm and write config? [y/N]" + footer
	case modeDone:
		return ""
	default:
		return header + "unknown state" + footer
	}
}

func (m model) reviewView() string {
	ans := m.toAnswers()
	var serviceKeys []string
	for k, v := range ans.Services {
		if v {
			serviceKeys = append(serviceKeys, k)
		}
	}
	sort.Strings(serviceKeys)

	return fmt.Sprintf(
		"Summary\nProject: %s\nStage: %s\nDomain: %s\nGit URL: %s\nHelm Repo: %s\nEnabled Services (%d): %s",
		maskEmpty(ans.ProjectName),
		maskEmpty(ans.ProjectStage),
		maskEmpty(ans.DomainName),
		maskEmpty(ans.ArgocdGitHttpsURL),
		maskEmpty(ans.ArgocdHelmRepoURL),
		len(serviceKeys),
		strings.Join(serviceKeys, ", "),
	)
}

func (m model) toAnswers() Answers {
	values := make(map[string]string, len(m.fields))
	for _, f := range m.fields {
		values[f.TargetName] = strings.TrimSpace(f.Value)
	}
	services := make(map[string]bool, len(m.serviceOptions))
	for _, opt := range m.serviceOptions {
		services[opt.Key] = opt.Enabled
	}

	return Answers{
		ProjectName:                 values["PROJECT_NAME"],
		ProjectStage:                values["PROJECT_STAGE"],
		DockerconfigBase64:          values["DOCKERCONFIG_BASE64"],
		ArgocdWizardAccountPassword: values["ARGOCD_WIZARD_ACCOUNT_PASSWORD"],
		ArgocdGitHttpsURL:           values["ARGOCD_GIT_HTTPS_URL"],
		ArgocdGitPatOrPassword:      values["ARGOCD_GIT_PAT_OR_PASSWORD"],
		ArgocdGitUsername:           values["ARGOCD_GIT_USERNAME"],
		DomainName:                  values["DOMAIN_NAME"],
		ArgocdHelmRepoURL:           values["ARGOCD_HELM_REPO_URL"],
		ArgocdHelmRepoUsername:      values["ARGOCD_HELM_REPO_USERNAME"],
		ArgocdHelmRepoPassword:      values["ARGOCD_HELM_REPO_PASSWORD"],
		Services:                    services,
	}
}

func maskEmpty(v string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "(empty)"
	}
	return trimmed
}

func (m model) questionProgress() float64 {
	if len(m.fields) == 0 {
		return 0
	}
	step := m.fieldIndex
	if step < 0 {
		step = 0
	}
	if step > len(m.fields) {
		step = len(m.fields)
	}
	return float64(step) / float64(len(m.fields))
}
