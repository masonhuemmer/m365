Feature: meetings consent
  Scenario: a plain login does not grant meetings
    Given the CLI is available
    When I sign in for meetings with a plain login
    Then the command succeeds
    And auth status reports meetings false

  Scenario: the meetings login grants meetings
    Given the CLI is available
    When I sign in for meetings with the meetings login
    Then the command succeeds
    And auth status reports meetings true

  Scenario: a meetings verb without meetings consent is refused
    Given a meetings fake without meetings consent
    When I save the transcript by event
    Then exit code 4
    And no transcript file is written
