Feature: meetings refusals
  Scenario: the tenant turned off Graph access to transcripts
    Given a meetings tenant with Graph access to transcripts turned off
    When I list transcripts by event
    Then exit code 4
    And the error names GraphAccessToTranscriptsDisabled

  Scenario: the tenant turned off speaker attribution
    Given a meetings tenant with speaker attribution turned off
    When I save the transcript by event
    Then exit code 4
    And the error names SpeakerAttributionNotAllowed
    And no transcript file is written
