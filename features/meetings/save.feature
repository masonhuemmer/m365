Feature: meetings save
  Scenario: save a transcript by event
    Given a meetings fake with one transcript
    When I save the transcript by event
    Then the command succeeds
    And the saved file equals the transcript fixture
    And the summary lists the fixture speakers
    And no transcript text is printed

  Scenario: save a transcript by join URL
    Given a meetings fake with one transcript
    When I save the transcript by join URL
    Then the command succeeds
    And the saved file equals the transcript fixture

  Scenario: save one of several transcripts by id
    Given a meetings fake with two transcripts
    When I save the transcript by event with id t2
    Then the command succeeds
    And the summary names transcript t2

  Scenario: an existing file is refused without overwrite
    Given a meetings fake with one transcript
    And the destination file already exists
    When I save the transcript by event
    Then exit code 3
    And the destination file is unchanged

  Scenario: overwriting a shared file leaves it owner-only
    Given a meetings fake with one transcript
    And the destination file already exists readable by others
    When I save the transcript by event with overwrite
    Then the command succeeds
    And the saved file is owner-only

  Scenario: a transcript without speaker names is refused
    Given a meetings fake whose transcript has no speaker names
    When I save the transcript by event
    Then exit code 5
    And no transcript file is written
