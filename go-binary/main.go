package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"kubara/cmd"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	AppName        = "kubara"
	defaultEnvFile = ".env"
)

var authors = []any{
	mail.Address{
		Name:    "Alexander Hoeft",
		Address: "alexander.hoeft@iits-consulting.de",
	},
	mail.Address{
		Name:    "Artem Lajko",
		Address: "artem.lajko@iits-consulting.de",
	},
	mail.Address{
		Name:    "Matthias Huether",
		Address: "matthias.huether@iits-consulting.de",
	},
	mail.Address{
		Name:    "Fabian Schmitt",
		Address: "fabian-patrice.schmitt@stackit.cloud",
	},
}

var (
	version                    = "dev" //version is dynamically set at build time via ldflags by GoReleaser. Defaults to "dev" for local builds.
	defaultTargetFilename      = "config"
	defaultManagedCatalog      = "managed-service-catalog"
	defaultOverlayValues       = "customer-service-catalog/helm/CLUSTER/argo-cd/values.yaml"
	singleTemplatePath         = "config.yaml.initial"
	defaultControlPlaneSecrets = "secrets-controlplane.yaml"
	defaultWorkerSecrets       = "secrets-worker.yaml"
	defaultSchemaPath          = "config.schema.json"
	argocdNamespace            = "argocd"
	externalSecretsNamespace   = "external-secrets"
)

//go:embed config.schema.json
var embeddedSchema []byte

//go:embed all:templates
var templatesFS embed.FS

//go:embed secrets-controlplane.yaml
var embeddedSecretControlPlane []byte

//go:embed secrets-worker.yaml
var embeddedSecretWorker []byte

func init() {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: zerolog.TimeFieldFormat,
		},
	)
}

func printEmbeddedFiles() {
	fmt.Println("Embedded files:")
	fmt.Println(" - config.schema.json")
	fmt.Printf(" - %s\n", singleTemplatePath)
	fmt.Printf(" - %s\n", defaultControlPlaneSecrets)
	fmt.Printf(" - %s\n", defaultWorkerSecrets)
	_ = fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("   [error reading %s: %v]\n", path, err)
			return nil
		}
		if !d.IsDir() {
			fmt.Printf(" - %s\n", path)
		}
		return nil
	})
}

func toJSONCompatible(i interface{}) interface{} {
	switch x := i.(type) {
	case map[string]interface{}:
		m2 := make(map[string]interface{}, len(x))
		for k, v := range x {
			m2[k] = toJSONCompatible(v)
		}
		return m2
	case []interface{}:
		a2 := make([]interface{}, len(x))
		for idx, v := range x {
			a2[idx] = toJSONCompatible(v)
		}
		return a2
	default:
		return x
	}
}

func validateConfig(config map[string]interface{}, schemaPath string) error {
	var schemaLoader gojsonschema.JSONLoader
	if schemaPath == defaultSchemaPath {
		schemaLoader = gojsonschema.NewBytesLoader(embeddedSchema)
	} else {
		abs, err := filepath.Abs(schemaPath)
		if err != nil {
			return fmt.Errorf("cannot resolve schema file: %v", err)
		}
		schemaLoader = gojsonschema.NewReferenceLoader("file://" + abs)
	}
	norm := toJSONCompatible(config)
	jb, err := json.Marshal(norm)
	if err != nil {
		return fmt.Errorf("cannot marshal config to JSON: %v", err)
	}
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewBytesLoader(jb))
	if err != nil {
		return fmt.Errorf("schema validation error: %v", err)
	}
	if !result.Valid() {
		var errs []string
		for _, e := range result.Errors() {
			errs = append(errs, e.String())
		}
		return fmt.Errorf("config does not conform to schema: %s", strings.Join(errs, "; "))
	}
	return nil
}

func loadDotEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read env file %s: %w", path, err)
	}
	env := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		env[k] = v
	}
	return env, nil
}

func loadYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func testConnection(kubeconfig string) {
	kc := kubeconfig
	if kc == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("home dir")
		}
		kc = filepath.Join(home, ".kube", "config")
	}
	log.Info().Msg("listing namespaces via kubectl…")
	execOrFatal(
		"kubectl",
		"--kubeconfig", kc,
		"get", "namespaces",
	)
}

func execOrFatal(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debug().Str("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))).Msg("executing")
	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err).Msgf("%s failed", name)
	}
}

func applyCRDs(kubeconfig, release, chart, values string) {
	helm := exec.Command(
		"helm", "template", "--include-crds",
		release, chart,
		"--values", values,
	)
	out, err := helm.Output()
	if err != nil {
		log.Fatal().Err(err).Msgf("helm template %s", release)
	}
	var buf bytes.Buffer
	for _, doc := range bytes.Split(out, []byte("\n---\n")) {
		if bytes.Contains(doc, []byte("kind: CustomResourceDefinition")) {
			buf.Write(doc)
			buf.WriteString("\n---\n")
		}
	}
	ku := exec.Command(
		"kubectl", "--kubeconfig", kubeconfig,
		"apply",
		"--server-side",
		"--force-conflicts",
		"-f", "-",
	)
	ku.Stdin = &buf
	ku.Stdout = os.Stdout
	ku.Stderr = os.Stderr
	log.Info().Str("release", release).Msg("applying CRDs")
	if err := ku.Run(); err != nil {
		log.Fatal().Err(err).Msg("kubectl apply CRDs")
	}
}

func waitPod(cs *kubernetes.Clientset, ns, sel string) {
	for {
		ps, err := cs.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{LabelSelector: sel})
		if err == nil && len(ps.Items) > 0 &&
			ps.Items[0].Status.ContainerStatuses != nil &&
			ps.Items[0].Status.ContainerStatuses[0].Ready {
			log.Info().Str("pod", sel).Msg("ready")
			return
		}
		log.Info().Str("pod", sel).Msg("waiting...")
		time.Sleep(5 * time.Second)
	}
}

func applyWorkerSecretsOnly(kubeconfig string, envMap map[string]string, managedCatalog string, overlayValuesPath string) {
	kc := kubeconfig
	if kc == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("home dir")
		}
		kc = filepath.Join(home, ".kube", "config")
	}

	if err := exec.Command("kubectl", "--kubeconfig", kc, "get", "ns", externalSecretsNamespace).Run(); err != nil {
		log.Info().Msgf("Creating namespace %s", externalSecretsNamespace)
		execOrFatal("kubectl", "--kubeconfig", kc, "create", "namespace", externalSecretsNamespace)
	}

	outWorker := os.Expand(string(embeddedSecretWorker), func(key string) string {
		if v, ok := envMap[key]; ok {
			return v
		}
		return os.Getenv(key)
	})
	cmd := exec.Command("kubectl", "--kubeconfig", kc, "replace", "--force", "-f", "-")
	cmd.Stdin = strings.NewReader(outWorker)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info().Msg("replacing worker secret only")
	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err).Msg("kubectl replace worker secret")
	}

	execOrFatal("helm", "repo", "add", "external-secrets", "https://charts.external-secrets.io")
	execOrFatal("helm", "dependency", "build", fmt.Sprintf("%s/helm/external-secrets/", managedCatalog))
	execOrFatal("helm", "repo", "add", "prometheus-community", "https://prometheus-community.github.io/helm-charts")
	execOrFatal("helm", "dependency", "build", fmt.Sprintf("%s/helm/kube-prometheus-stack/", managedCatalog))

	applyCRDs(kc, "external-secrets", fmt.Sprintf("%s/helm/external-secrets/", managedCatalog), overlayValuesPath)
	applyCRDs(kc, "kube-prometheus-stack", fmt.Sprintf("%s/helm/kube-prometheus-stack/", managedCatalog), overlayValuesPath)
}

func constructOverlayValuesPath(configFile, schemaFile, overlayValuesPattern string) string {
	cfgMap, err := loadYAML(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	if err := validateConfig(cfgMap, schemaFile); err != nil {
		log.Fatal().Err(err).Msg("schema validation failed")
	}

	clustersI, ok := cfgMap["clusters"].([]interface{})
	if !ok || len(clustersI) == 0 {
		log.Fatal().Msg("no clusters defined in config")
	}
	firstCluster, ok := clustersI[0].(map[string]interface{})
	if !ok {
		log.Fatal().Msg("invalid cluster entry")
	}
	clusterName := fmt.Sprint(firstCluster["name"])

	return strings.Replace(overlayValuesPattern, "CLUSTER", clusterName, 1)
}

func bootstrapArgocd(
	kubeconfig, configFile, schemaFile, managedCatalog, overlayValuesPattern string,
	withES, withProm bool,
	envMap map[string]string,
) {
	kc := kubeconfig
	if kc == "" {
		home, _ := os.UserHomeDir()
		kc = filepath.Join(home, ".kube", "config")
	}

	overlayValuesPath := constructOverlayValuesPath(configFile, schemaFile, overlayValuesPattern)

	execOrFatal("helm", "repo", "add", "argo", "https://argoproj.github.io/argo-helm")
	execOrFatal("helm", "dependency", "build", fmt.Sprintf("%s/helm/argo-cd/", managedCatalog))
	if withES {
		execOrFatal("helm", "repo", "add", "external-secrets", "https://charts.external-secrets.io")
		execOrFatal("helm", "dependency", "build", fmt.Sprintf("%s/helm/external-secrets/", managedCatalog))
	}
	if withProm {
		execOrFatal("helm", "repo", "add", "prometheus-community", "https://prometheus-community.github.io/helm-charts")
		execOrFatal("helm", "dependency", "build", fmt.Sprintf("%s/helm/kube-prometheus-stack/", managedCatalog))
	}

	applyCRDs(kc, "argocd", fmt.Sprintf("%s/helm/argo-cd/", managedCatalog), overlayValuesPath)
	if withES {
		applyCRDs(kc, "external-secrets", fmt.Sprintf("%s/helm/external-secrets/", managedCatalog), overlayValuesPath)
	}
	if withProm {
		applyCRDs(kc, "kube-prometheus-stack", fmt.Sprintf("%s/helm/kube-prometheus-stack/", managedCatalog), overlayValuesPath)
	}

	pw, ok := envMap["ARGOCD_WIZARD_ACCOUNT_PASSWORD"]
	if !ok {
		log.Fatal().Msg("ARGOCD_WIZARD_ACCOUNT_PASSWORD not in env")
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		log.Fatal().Err(err).Msg("bcrypt.GenerateFromPassword")
	}
	hashed := string(hashBytes)

	helmArgs := []string{
		"template", "argocd", fmt.Sprintf("%s/helm/argo-cd/", managedCatalog),
		"--values", overlayValuesPath,
		"--api-versions=monitoring.coreos.com/v1",
		"--set", fmt.Sprintf("argo-cd.configs.secret.extra.accounts\\.wizard\\.password=%s", hashed),
		"--namespace", argocdNamespace,
	}
	helmCmd := exec.Command("helm", helmArgs...)
	kuCmd := exec.Command("kubectl", "--kubeconfig", kc, "apply", "--server-side", "--force-conflicts", "-f", "-")
	r, w := io.Pipe()
	helmCmd.Stdout = w
	kuCmd.Stdin = r
	helmCmd.Stderr = os.Stderr
	kuCmd.Stdout = os.Stdout
	kuCmd.Stderr = os.Stderr

	log.Info().Msg("bootstrapping ArgoCD")
	if err := helmCmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("helm start")
	}
	if err := kuCmd.Start(); err != nil {
		log.Fatal().Err(err).Msg("kubectl start")
	}
	if err = helmCmd.Wait(); err != nil {
		log.Err(err)
	}
	if err = w.Close(); err != nil {
		log.Err(err)
	}
	if err = kuCmd.Wait(); err != nil {
		log.Err(err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		log.Fatal().Err(err).Msg("kubeconfig")
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("clientset")
	}
	waitPod(cs, argocdNamespace, "app.kubernetes.io/name=argocd-server")
	waitPod(cs, argocdNamespace, "app.kubernetes.io/name=argocd-repo-server")

	if _, err := cs.CoreV1().Namespaces().Get(context.Background(), externalSecretsNamespace, metav1.GetOptions{}); err != nil {
		log.Info().Msg("creating namespace external-secrets")
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: externalSecretsNamespace}}
		if _, err := cs.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{}); err != nil {
			log.Fatal().Err(err).Msg("failed to create namespace external-secrets")
		}
	}

	outSec := os.Expand(string(embeddedSecretControlPlane), func(key string) string {
		if v, ok := envMap[key]; ok {
			return v
		}
		return os.Getenv(key)
	})
	kuReplace := exec.Command("kubectl", "--kubeconfig", kc, "replace", "--force", "-f", "-")
	kuReplace.Stdin = strings.NewReader(outSec)
	kuReplace.Stdout = os.Stdout
	kuReplace.Stderr = os.Stderr
	log.Info().Msg("replacing ArgoCD secret")
	if err := kuReplace.Run(); err != nil {
		log.Fatal().Err(err).Msg("kubectl replace")
	}

	dns := os.Getenv("DNS_NAME")
	if v, ok := envMap["DNS_NAME"]; ok && v != "" {
		dns = v
	}
	ingressMsg := ""
	if dns != "" {
		ingressMsg = fmt.Sprintf(" or try: http://%s/argocd (if ingress is running)", dns)
	}

	log.Info().Msgf(`ArgoCD bootstrap complete!
You can access the Argo CD UI with user "wizard" and your chosen password "%s" at:

    kubectl port-forward svc/argocd-server -n argocd 8080:443 --kubeconfig ...

Then open: http://localhost:8080/argocd%s`, pw, ingressMsg)
}

var (
	configFile          string
	schemaFile          string
	workDir             string
	kubeconfigFilePath  string
	managedCatalog      string
	overlayValues       string
	filename            string
	listEmbedded        bool
	testK8sConnection   bool
	createWorkerSecrets bool
	bootstrap           bool
	withProm            bool
	withES              bool
	base64Mode          bool
	encodeFlag          bool
	decodeFlag          bool
	inputFile           string
	inputString         string
	envFile             string
)

func NewAppAction(cmd *cli.Command) error {
	if kubeconfigFilePath == "~/.kube/config" {
		if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
			kubeconfigFilePath = envKC
		}
	}
	// If base64 utility mode is enabled, handle it here and exit
	if base64Mode {
		if (encodeFlag && decodeFlag) || (!encodeFlag && !decodeFlag) {
			return cli.Exit("Error: specify either --encode or --decode", 1)
		}
		if (inputString != "" && inputFile != "") || (inputString == "" && inputFile == "") {
			return cli.Exit("Error: specify exactly one of --string or --file", 1)
		}
		var data []byte
		var err error
		if inputFile != "" {
			data, err = os.ReadFile(inputFile)
			if err != nil {
				log.Fatal().Err(err).Msgf("Cannot read file: %s", inputFile)
				return cli.Exit("Error: reading file", 1)
			}
		} else {
			data = []byte(inputString)
		}
		if encodeFlag {
			fmt.Print(base64.StdEncoding.EncodeToString(data))
		} else {
			decoded, err := base64.StdEncoding.DecodeString(string(data))
			if err != nil {
				log.Fatal().Err(err).Msg("Invalid base64 input")
				return cli.Exit("Error: invalid base64 input", 1)
			}
			_, err = os.Stdout.Write(decoded)
			if err != nil {
				return cli.Exit("Error: writing decoded base64 input", 1)
			}
		}
		return nil
	}

	if listEmbedded {
		printEmbeddedFiles()
		return nil
	}

	if cmd.NumFlags() == 0 {
		cli.ShowAppHelpAndExit(cmd, 0)
	}

	cwd, _ := os.Getwd()
	if workDir == "" {
		workDir = cwd
	}
	if configFile == "" {
		configFile = filepath.Join(workDir, "config.yaml")
	}

	envPath := envFile
	if !filepath.IsAbs(envPath) {
		envPath = filepath.Join(cwd, envPath)
	}
	envMap, err := loadDotEnvFile(envPath)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Error: %v", err), 1)
	}

	if createWorkerSecrets && !bootstrap {
		overlayValuesPath := constructOverlayValuesPath(configFile, schemaFile, overlayValues)
		applyWorkerSecretsOnly(kubeconfigFilePath, envMap, managedCatalog, overlayValuesPath)
		return nil
	}

	switch {
	case testK8sConnection:
		testConnection(kubeconfigFilePath)
	case bootstrap:
		bootstrapArgocd(
			kubeconfigFilePath,
			configFile,
			schemaFile,
			managedCatalog,
			overlayValues,
			withES,
			withProm,
			envMap,
		)
	default:
		cli.ShowAppHelpAndExit(cmd, 0)
	}
	return nil
}

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"c"},
			Value:       "config.yaml",
			Usage:       "Path to the configuration file",
			Destination: &configFile,
		},
		&cli.StringFlag{
			Name:        "schema-file",
			Value:       defaultSchemaPath,
			Usage:       "Path to JSON Schema for config.yaml",
			Destination: &schemaFile,
		},
		&cli.StringFlag{
			Name:        "work-dir",
			Aliases:     []string{"w"},
			Value:       ".",
			Usage:       "Working directory",
			Destination: &workDir,
		},
		&cli.StringFlag{
			Name:        "kubeconfig",
			Value:       "~/.kube/config",
			Usage:       "Path to kubeconfig file",
			Destination: &kubeconfigFilePath,
		},
		&cli.StringFlag{
			Name:        "managed-catalog",
			Value:       defaultManagedCatalog,
			Usage:       "Helm chart path prefix. Folder name in which the managed catalog is stored relative to workdir",
			Destination: &managedCatalog,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Value:       defaultOverlayValues,
			Usage:       "Path pattern to ArgoCD values.yaml; must include CLUSTER",
			Destination: &overlayValues,
		},
		&cli.StringFlag{
			Name:        "filename",
			Value:       defaultTargetFilename,
			Usage:       "Basename for the generated file",
			Destination: &filename,
		},
		&cli.BoolFlag{
			Name:        "list-embedded",
			Value:       false,
			Usage:       "List embedded files (schema, initial template, templates/, secrets) and exit",
			Destination: &listEmbedded,
		},
		&cli.StringFlag{
			Name:        "env-file",
			Value:       defaultEnvFile,
			Usage:       "Path to the .env file",
			Destination: &envFile,
		},
		&cli.BoolFlag{
			Name:        "test-connection",
			Value:       false,
			Usage:       "Check if Kubernetes cluster can be reached. List namespaces and exit",
			Destination: &testK8sConnection,
		},
		&cli.BoolFlag{
			Name:        "create-secrets-worker",
			Value:       false,
			Usage:       "also create/apply the worker secret",
			Destination: &createWorkerSecrets,
		},
		&cli.BoolFlag{
			Name:        "bootstrap-argocd",
			Value:       false,
			Usage:       "Perform full ArgoCD bootstrap",
			Destination: &bootstrap,
		},
		&cli.BoolFlag{
			Name:        "with-es-crds",
			Value:       false,
			Usage:       "Also install external-secrets CRDs",
			Destination: &withES,
		},
		&cli.BoolFlag{
			Name:        "with-prometheus-crds",
			Value:       false,
			Usage:       "Also install kube-prometheus-stack CRDs",
			Destination: &withProm,
		},
		&cli.BoolFlag{
			Name:        "base64",
			Value:       false,
			Usage:       "Enable base64 encode/decode mode",
			Destination: &base64Mode,
		},
		&cli.BoolFlag{
			Name:        "encode",
			Value:       false,
			Usage:       "Base64 encode input",
			Destination: &encodeFlag,
		}, &cli.BoolFlag{
			Name:        "decode",
			Value:       false,
			Usage:       "Base64 decode input",
			Destination: &decodeFlag,
		},
		&cli.StringFlag{
			Name:        "string",
			Value:       "",
			Usage:       "Input string for base64 operation",
			Destination: &inputString,
		},
		&cli.StringFlag{
			Name:        "file",
			Value:       "",
			Usage:       "Input file path for base64 operation",
			Destination: &inputFile,
		},
	}

	app := &cli.Command{
		Name:        AppName,
		Version:     version,
		Authors:     authors,
		Copyright:   "",
		Usage:       "",
		Flags:       flags,
		UsageText:   "",
		Description: "",
		Commands: []*cli.Command{
			cmd.NewInitCmd(),
			cmd.NewGenerateCmd(),
		},
		Action: func(cCtx context.Context, cmd *cli.Command) error {
			return NewAppAction(cmd)
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error running program")
	}

}
