Feature: meetings list
  Scenario: list by event shows only that occurrence
    Given a meetings list fake with transcripts across two occurrences
    When I list transcripts by event
    Then the command succeeds
    And the listed transcripts are t-in-b t-in-a

  Scenario: list by join URL shows every transcript newest first
    Given a meetings list fake with transcripts across two occurrences
    When I list transcripts by join URL
    Then the command succeeds
    And the listed transcripts are t-in-b t-in-a t-prev

  Scenario: a meeting without transcripts lists none
    Given a meetings list fake with no transcripts
    When I list transcripts by event
    Then the command succeeds
    And the listed count is 0
