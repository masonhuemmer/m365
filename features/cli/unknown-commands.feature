Feature: Unknown commands are usage errors that name the word
  A namespace or verb that m365 does not know exits with the usage code and
  writes one JSON error to stderr whose message names the unknown word.

  Scenario Outline: an unknown namespace or verb is a usage error that names it
    Given the CLI is available
    When I run "<args>"
    Then exit code <code>
    And the error is <class>: <message>

    Examples:
      | args            | code | class | message                        |
      | frobnicate      | 3    | usage | unknown namespace "frobnicate" |
      | mail frobnicate | 3    | usage | unknown mail verb "frobnicate" |
      | auth frob       | 3    | usage | unknown auth verb "frob"       |
