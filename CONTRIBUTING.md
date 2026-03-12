# Contributing to Kubara

Thank you for your interest in contributing to **Kubara**!
Whether you're fixing bugs, improving documentation, or adding features - all contributions are welcome.

## 🧑‍💻 Contributor vs. Maintainer

* **Contributors**: Anyone submitting code, docs or ideas via Issues or Pull Requests.
* **Maintainers**: Core team members with permission to review, approve, and merge contributions. Maintainers help enforce standards and ensure quality.
    * Maintainers: @alexander.hoeft @matthias.huether @felix.kammerer @artem.lajko

## 🐛 Reporting Issues

If you discover a bug or have a feature request, please open an issue in [Issues Tracker](https://kubara.git.onstackit.cloud/STACKIT/kubara/issues) and describe:

* What's happening
* Steps to reproduce
* Expected vs. actual result
* Logs or screenshots (if applicable)

If you are open an issue, a template will guide you through the process.

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

## 🧠 Branch Strategy

* `master`: Latest features - unstable, may change without notice
* `tag/vX.X.X-XX`: tags point to the latest stable version
* `<some-feat-branch>`: work on features

## 💬 Code Review Etiquette

We aim for respectful and constructive collaboration.
Please:

* Be open to feedback and iterative improvement
* Respond to review comments in a timely manner
* Avoid mixing unrelated changes in a single PR

> A good PR tells a story: *what's changing, why it matters, and how to review it.*

## Integration Requirements Catalogue

Are you missing a feature or you think that the software and community 
would profit greatly if this new tool is included? 
For that you can propose a new tool to be included in Kubara.
In order to deliver the best possible software and to make maintenance easier for us and the community we have created a requirements catalogue
that is required for any serious proposal.
The catalogue is split into different parts, the numbers at the end indicate how many percentages they contribute to a proposal to be included.
More on that in subchapter 5.

The following points describe based on what criteria the tool will be rated.

## Strategic Alignment Criteria (40%)
These requirements ensure that the tool to be proposed fit the current and future trajections of the project.

### 1.1 Vision Alignment (Weight: 25%)
- **Core Mission Fit**: Tool directly supports Kubara's primary objectives
- **Community Alignment**: Tool supports or enhances community-driven development
- **Long-term Sustainability**: Active maintenance, clear roadmap, healthy contributor base

### 1.2 Technical Integration (Weight: 15%)
- **Architecture Compatibility**: Minimal disruption to existing stack patterns
- **API Consistency**: Aligns with established integration patterns
- **Dependency Management**: No conflicting dependencies or version constraints
- **Resource Footprint**: Acceptable CPU/memory/storage requirements
- **Security and Best Practices**: Code validating, Image CVE scanning and linting by maintainers

## Operational Impact Assessment (30%)
These requirements look at how mature the proposed tool is and if it is not tech cruft.

### 2.1 Maintenance Overhead (Weight: 15%)
- **Update Frequency**: Reasonable release cycle (not excessive breaking changes)
- **Security Support**: Active security patches and vulnerability response
- **Documentation Quality**: Comprehensive, up-to-date documentation
- **Community Support**: Active issue resolution and community engagement
- **Maturity**: In which stage is the project (alpha, beta, GA)
- **Adoption Rate**: How well is this tool established?

### 2.2 Integration Complexity (Weight: 15%)
- **Implementation Effort**: High effort for initial integration
- **Migration Path**: Clear upgrade/downgrade procedures
- **Rollback Strategy**: Defined removal process with minimal impact

## Value Proposition Criteria (30%)
The following points look at if the tool proposed contribute something novel to the existing toolset in a way that leads to a net gain.

### 3.1 Problem Solving (Weight: 20%)
- **Pain Point Resolution**: Addresses specific user needs
- **Value Gains**: General improvement in workflows or capabilities
- **Feature Uniqueness**: Provides capabilities not available in current stack
- **User Demand**: Evidence of community/team need (issues, requests, surveys)

### 3.2 Cost-Benefit Analysis (Weight: 10%)
- **TCO Assessment**: Total cost of ownership is significantly lower than projected value
- **Learning Curve**: Effort for the team to achieve proficiency
- **Support Requirements**: No specialized skills or external dependencies
- **Scalability Impact**: Positive or neutral effect on system scalability

## Evaluation Process
What is required in the propoasl and how does the review look like?

### 4.1 Submission Requirements
- **Problem Statement**: Clear description of issue being solved
- **Alternative Analysis**: Comparison with existing solutions
- **Implementation Plan**: Technical approach and timeline
- **Success Metrics**: Defined KPIs for measuring impact

### 4.2 Review Stages
- **Initial Screening**: Basic compliance check
- **Technical Evaluation**: Architecture and integration analysis
- **Pilot Testing**: Limited scope implementation
- **Final Decision**: Governance committee approval

## Decision Matrix
Based on what percentages will a tool be included and on what criteria will the proposal fail.

### 5.1 Scoring Thresholds
- **Auto-Approve**: Score ≥ 85/100 with no critical failures
- **Conditional Approve**: Score 70-84/100 with mitigation plan
- **Reject**: Score < 70/100 or critical failure in any category

### 5.2 Critical Failure Conditions
- **Proprietary licensing or vendor lock-in**
- **Open Source Compliance**: 100% open-source with permissive license (MIT, Apache 2.0, BSD)
- **Security vulnerabilities without remediation path**
- **Breaking changes to core functionality**
- **> 100 hours integration effort**
- **No active maintenance for > 6 months**

## Rejection Criteria
The following is a list on critera which leads to the automatic rejection of the tool proposal.

### 6.1 Automatic Rejection
- **Duplicate functionality with existing tools**
- **Requires proprietary dependencies**
- **No clear migration path from current solutions**
- **Exceeds resource allocation for maintenance**

### 6.2 Rejection Response Template
- **Clear explanation of decision criteria**
- **Specific areas of non-compliance**
- **Suggestions for alternative approaches**
- **Path for resubmission**: Clear requirements for reconsideration

Support: [Support](support.md)
