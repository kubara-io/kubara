# Kubara Maintainers

More details about our governance and how maintainers are selected can be found below.

## Maintainers

| Maintainer       | E-Mail                                | Affiliation              |
|------------------|---------------------------------------|--------------------------|
| Artem Lajko      | artem.lajko@iits-consulting.de        | iits-consulting          |
| Matthias Huether | matthias.huether@iits-consulting.de   | iits-consulting          |
| Fabian Schmitt   | fabian-patrice.schmitt@digits.schwarz | STACKIT / Schwarz Digits |

## Governance

This document provides an initial governance structure for the Kubara project.
We've deliberately kept this as simple as possible for now, but
we expect the governance to evolve as the project grows.


## Team Members / Contributors

A Team Member is any member of the [kubara-team group](https://groups.google.com/g/tbd-team)
Google Group.

Team member status may be given to those who have made ongoing contributions to
the project for at least 3 months. Contributions can include
code improvements, notable work on documentation, organizing events,
user support, and other contributions.

New members may be proposed by any existing member by email to
[kubara-team group](https://groups.google.com/g/kubara-team).

It is highly desirable to reach consensus about acceptance of a new member.
However, the proposal is ultimately voted on by a formal 2/3 majority vote.

If the new member proposal is accepted, the proposed team member should be
contacted privately via email/chat to confirm or deny their acceptance of team
membership. This email will also be CC'd to kubara-team for record-keeping
purposes.

Team members may retire at any time by emailing the team.

Team members can be removed by 2/3 majority vote on the team mailing list. For
this vote, the member in question is not eligible to vote and does not count
towards the quorum. Any removal vote can cover only one single person.



## Maintainers

Kubara maintainers have write access to the Kubara Repository.
They can merge their own patches or patches from others. The current maintainers
can be found in above. Maintainers collectively manage the project's resources and contributors.

This privilege is granted with some expectation of responsibility: maintainers
are people who care about the Kubara project and want to help it grow and
improve. A maintainer is not just someone who can make changes, but someone who
has demonstrated their ability to collaborate with the team, get the most
knowledgeable people to review code and docs, contribute high-quality code, and
follow through to fix issues (in code or tests).

A maintainer is a contributor to the project's success and a citizen helping
the project succeed.

## Becoming a Maintainer

To become a Maintainer, you need to demonstrate the following:

* commitment to the project:
  * participate in discussions, contributions, code and documentation reviews
  * perform reviews for several non-trivial pull requests,
  * contribute to several non-trivial pull requests and have them merged,
* ability to write quality code and/or documentation,
* ability to collaborate with the team,
* understanding of how the team works (policies, processes for testing and code review, etc.),
* understanding of the project's code base and coding and documentation style.

A new maintainer must be proposed by an existing team member by sending an email to the
private mailing list (kubara-team@TBD). A 2/3 majority of team members
must vote to approve a new maintainer. After the new maintainer is approved, they will
be added to the private team member mailing list.

## Voting

While most business in Kubara is conducted by "lazy consensus", periodically
the Maintainers may need to vote on specific actions or changes.
A vote can be taken by emailing the private maintainer mailing list for sensitive
matters or by creating an issue to allow for public comment from the broader
community. Any Maintainer may demand a vote be taken.

Most votes require a simple majority of all Maintainers to succeed. Maintainers
can be removed by a 2/3 majority vote of all Maintainers, and changes to this
Governance require a 2/3 vote of all Maintainers.

## Commit Signing for Maintainers

We **strongly recommend** that all maintainers sign their commits.  
Signed commits verify your identity and help ensure that changes to the repository are authentic and have not been tampered with.

### Benefits of signed commits
- Verify the author's identity
- Prevent commit forgery or alteration
- Improve trust and security in the project's history

### How to start
1. Generate a GPG key pair on your local machine.
2. Upload your **public** GPG key in your profile settings under **SSH / GPG keys**.
3. Configure Git to use your GPG key for signing.
4. Sign commits using:
   ```bash
   git commit -S -m "Your commit message"
   ```
   Push your changes as usual - Git will show a "Verified" badge next to signed commits.

For a detailed step-by-step guide, see the [Codeberg GPG key documentation](https://docs.codeberg.org/security/gpg-key/).