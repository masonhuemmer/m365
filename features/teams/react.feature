Feature: teams react
  m365 teams react adds or removes an emoji reaction on one chat message, picked
  by chat and message ID or as the newest message someone else sent (--last).
  teams messages shows each message's reactions.

  Scenario: react help names the forms and the reaction names
    Given the CLI is available
    When I run "teams react --help"
    Then the command succeeds
    And stdout contains "like, heart, laugh, surprised, sad, angry, eyes"

  Scenario Outline: react to a message with an emoji
    Given the CLI is available
    When I run "teams react chat-ajay <message> <emoji>"
    Then the command succeeds
    And stdout JSON "message_id" is "<message>"
    And stdout JSON "reaction" is "<emoji>"
    And stdout JSON "reacted" is true

    Examples:
      | message     | emoji |
      | cmsg-ajay-1 | 🎉    |
      | cmsg-ajay-2 | 👀    |

  Scenario: a name gives its emoji
    Given the CLI is available
    When I run "teams react chat-ajay cmsg-ajay-1 like"
    Then the command succeeds
    And stdout JSON "reaction" is "👍"

  Scenario: --last reacts to the newest message someone else sent
    Given the CLI is available
    When I run "teams react --to Ajay --last eyes"
    Then the command succeeds
    And stdout JSON "chat_id" is "chat-ajay"
    And stdout JSON "message_id" is "cmsg-ajay-1"
    And stdout JSON "from" is "Ajay Kumar"
    And stdout JSON "reaction" is "👀"

  Scenario: --remove takes a reaction back
    Given the CLI is available
    When I run "teams react chat-ajay cmsg-ajay-2 like --remove"
    Then the command succeeds
    And stdout JSON "removed" is true

  Scenario: a dry run changes nothing
    Given the CLI is available
    When I run "teams react chat-ajay cmsg-ajay-1 like --dry-run"
    Then the command succeeds
    And stdout JSON "dry_run" is true
    And stdout JSON "action" is "react"
    And nothing was sent

  Scenario: text is not a reaction
    Given the CLI is available
    When I run "teams react chat-ajay cmsg-ajay-1 hello"
    Then exit code 3
    And the error is usage: reaction "hello" is not an emoji or one of: like, heart, laugh, surprised, sad, angry, eyes
    And nothing was sent

  Scenario: --to needs --last
    Given the CLI is available
    When I run "teams react --to Ajay like"
    Then exit code 3
    And the error is usage: --to needs --last; to pick another message, pass CHAT_ID MESSAGE_ID
    And nothing was sent

  Scenario: an unknown message is not found
    Given the CLI is available
    When I run "teams react chat-ajay cmsg-missing like"
    Then exit code 6

  Scenario: teams messages shows reactions with the reacting member's name
    Given the CLI is available
    When I run "teams messages chat-ajay"
    Then the command succeeds
    And stdout JSON "items.1.reactions.0.emoji" is "👍"
    And stdout JSON "items.1.reactions.0.user_id" is "u-ajay"
    And stdout JSON "items.1.reactions.0.user" is "Ajay Kumar"
