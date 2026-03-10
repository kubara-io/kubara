# Contributing to Kubara

Thank you for your interest in contributing to **Kubara**!
Whether you're fixing bugs, improving documentation, or adding features - all contributions are welcome.

## 🧑‍💻 Contributor vs. Maintainer

* **Contributors**: Anyone submitting code, docs or ideas via Issues or Pull Requests.
* **Maintainers**: Core team members with permission to review, approve, and merge contributions. Maintainers help enforce standards and ensure quality.
  [Current Maintainers](maintainers.md)
---

## 🐛 Reporting Issues

If you discover a bug or have a feature request, please open an issue in [Issues Tracker](https://kubara.git.onstackit.cloud/STACKIT/kubara/issues) and describe:

* What's happening
* Steps to reproduce
* Expected vs. actual result
* Logs or screenshots (if applicable)

If you are open an issue, a template will guide you through the process.

---

## 🚀 How to Contribute

### Before You Start Working on a Bug or Feature

Before you begin working on a bug fix or implementing a new feature, please create an issue or feature request first (see above). 
This allows us to briefly discuss the best approach to solving the problem and avoid duplicated efforts.

For larger topics, such as fundamental or strategic decisions, we recommend discussing them in a contributor meeting or during the Kubara Office Hours.
For significant technical decisions, please document the outcome using an Architecture Decision Record (ADR), see [ADR](../7_decisions/ADR.md).
For more information, please refer to our support documentation: [Support](support.md)

### Preparations: Pre-commit Hooks

We use pre-commit hooks to enforce coding standards and maintain code quality across the project.
If you plan to contribute, please make sure to install and configure the hooks locally as well. They will help you adhere to the required standards before code is committed, ensuring a smoother development process.
These hooks are also executed in the CI pipeline, and any violations will cause the pipeline to fail. So even if you bypass them locally, your code will not be accepted unless it passes all checks.
You can find installation instructions here: https://kubara.git.onstackit.cloud/STACKIT/kubara/src/branch/master/go-binary/README.md

Once you have set up the pre-commit hooks, you can follow the steps below to start contributing:

1. **Check if an ADR is required**: If your change involves a significant technical or architectural decision, create an Architecture Decision Record (ADR) first, see [ADR](../7_decisions/ADR.md)
2. **Fork** the repository and clone it locally, see also here: https://forgejo.org/docs/latest/user/pull-requests-and-git-flow/
2. **Create a new branch** for your work
3. **Implement your changes**
4. **Run checks** before submitting
5. **Commit** using [Conventional Commits](https://www.conventionalcommits.org)
6. **Open a Pull Request** to the `dev` branch () -> Please note the chapter: Pull Requests: Conventions & Best Practices

### 🧩 Pull Requests: Conventions & Best Practices

#### 📦 One PR per Topic

Avoid bundling multiple unrelated changes (e.g. fixing unrelated bugs or adding a bug fix and a new feature) in a single PR. Instead, create a separate PR for each topic.

This approach helps to:

    Keep reviews focused and easier to manage
    Write clear and meaningful PR titles
    Improve the clarity of commit history and changelogs
    Minimize risk when reverting changes
    Small, focused PRs are easier to review, less prone to merge conflicts, and lead to a more maintainable codebase.

#### 🔤 PR Title Naming Convention

PR titles should follow the structure of Conventional Commits, aligned with the main type of change introduced:

    feat: for new features
    fix: for bug fixes
    docs: for documentation changes
    refactor: for internal code improvements
    chore: for maintenance, tooling, or CI-related updates

Examples:

    feat: add password reset functionality
    fix: handle null user session on login
    docs: improve README with setup instructions

Keep it short and descriptive. Use a scope in parentheses if needed (e.g. fix(auth): ...).

#### 📝 PR Description Requirements

A Pull Request template is automatically loaded when you open a new PR.
Please fill it out completely and thoughtfully - it's there to help reviewers understand:

    What your change does
    Why it's needed
    How it was implemented
    Any relevant issues or tickets
    Special notes for testing, review, or deployment

Well-written descriptions lead to faster reviews and fewer misunderstandings.
Do not leave the template empty or remove sections without reason - each part serves a purpose.

---

## 🧠 Branch Strategy

* `master`: Latest features - unstable, may change without notice
* `tag/vX.X.X-XX`: tags point to the latest stable version
* `<some-feat-branch>`: work on features

---

## 💬 Code Review Etiquette

We aim for respectful and constructive collaboration.
Please:

* Be open to feedback and iterative improvement
* Respond to review comments in a timely manner
* Avoid mixing unrelated changes in a single PR

> A good PR tells a story: *what's changing, why it matters, and how to review it.*

---

Support: [Support](support.md)
