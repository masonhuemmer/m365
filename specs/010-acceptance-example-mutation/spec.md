# Feature Specification: Acceptance-example mutation gate

**Feature Branch**: `010-acceptance-example-mutation`

**Created**: 2026-11-11

**Status**: Draft

**Input**: Retire exception EX-EC-002 (`specs/001-m365-cli/plan.md`). The constitution requires an Acceptance-Pipeline-Specification (APS) flow that ends in example mutation, and a blocking `acceptance-example mutation` target. Today the target mutates nothing: no feature has an Examples table, the acceptance runtime rejects one, and there is no runner worker or generator metadata for the APS `gherkin-mutator`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - The mutation gate catches an example nothing checks (Priority: P1)

A maintainer runs `make acceptance-mutation`. For each feature with an Examples table, the APS mutator changes one example cell at a time and runs that feature's generated acceptance test against the changed data. A change that no assertion notices survives, and the target fails and names it.

**Why this priority**: It is the gate the constitution requires; without it, example data in a feature can drift from what the tests check.

**Independent Test**: Run `make acceptance-mutation` on the branch: every mutation of `features/cli/unknown-commands.feature` is killed and the target exits 0. Add a column that no step reads to a scratch copy of the feature: the target exits non-zero and reports the surviving mutation.

**Acceptance Scenarios**:

1. **Given** a feature with an Examples table whose every cell is asserted, **When** the maintainer runs the target, **Then** every mutation is killed, the target exits 0, and `features/` is unchanged.
2. **Given** a cell that no step checks, **When** the target runs, **Then** that mutation survives and the target exits non-zero.
3. **Given** no feature with an Examples table, **When** the target runs, **Then** it exits non-zero and says there is nothing to mutate.
4. **Given** a mutation run that cannot start or finish a test (missing binary, timeout, no test selected), **When** the target runs, **Then** the mutation is an error, never a pass.

### User Story 2 - Features can use Scenario Outlines, Examples and Background (Priority: P1)

An author writes a Scenario Outline with an Examples table, or a Background. The generated acceptance test runs the scenario once per example row, with each `<placeholder>` replaced by the row's value, and runs the Background steps first.

**Why this priority**: The mutator only mutates example values, so the runtime must run them.

**Independent Test**: The unknown-commands outline runs one subtest per row; a placeholder with no column fails the test before any step runs.

**Acceptance Scenarios**:

1. **Given** an outline with N example rows, **When** its generated test runs, **Then** N executions run, each with its row's values.
2. **Given** a feature with a Background, **When** a scenario runs, **Then** the Background steps run first, in a fresh world.
3. **Given** a step whose `<name>` has no column in the row, **When** the test runs, **Then** it fails and names the placeholder.
4. **Given** any expanded step with no handler, **When** the test runs, **Then** it fails before any scenario runs and lists every such step (as today).

### User Story 3 - Unknown commands are stated as examples (Priority: P2)

`features/cli/unknown-commands.feature` lists, per row, an unknown namespace or verb, its exit code, and the JSON error's class and message, which name the unknown word. Each cell is asserted, so each mutation is caught.

**Acceptance Scenarios**:

1. **Given** the fake CLI, **When** each row's command runs, **Then** its exit code and its error's class and message match the row.

### Edge Cases

- Existing features run exactly as before (same scenarios, same steps, same names).
- The runtime reads the parser's IR, never the `.feature` text, so comments such as the mutation manifest do not affect a run.
- `M365_ACCEPTANCE_IR` replaces the IR only for the test the worker selects.

## Requirements *(mandatory)*

- **FR-001**: Generated acceptance tests MUST run the parser IR of their feature, including Examples rows and Background.
- **FR-002**: The runtime MUST substitute `<name>` placeholders from the current example row and MUST fail a missing value.
- **FR-003**: The runtime MUST fail every unsupported (unmatched) step before running a scenario, listing all of them.
- **FR-004**: The generator MUST write APS metadata for each feature (schema_version 1, feature_path, ir_path, implementation_hash over generated files only, hash_scope, generated_files).
- **FR-005**: A persistent runner worker MUST implement the APS newline-delimited JSON protocol and classify outcomes as test_success, test_failure or infrastructure_error; it MUST write nothing but protocol lines to stdout.
- **FR-006**: `make acceptance-mutation` MUST run `gherkin-mutator` on every feature with an Examples table and MUST fail on a survivor, an error, or no feature to mutate.
- **FR-007**: Exception EX-EC-002 MUST be removed from the 001 plan (a one-line retirement note may stay) once the gate passes.
- **FR-008**: No locked test is edited.

## Success Criteria *(mandatory)*

- **SC-001**: `make acceptance-mutation` exits 0 with every mutation killed, and non-zero for a planted surviving cell.
- **SC-002**: `make verify` passes, and `make acceptance` runs the same scenarios as before plus the new outline rows.
- **SC-003**: The 001 plan has no EX-EC-002 exception section.

## Assumptions

- The gate stays outside `make verify`, as today; it runs before a release and after acceptance-contract changes.
- The gate runs every mutation each time (APS level `full`) on a copy of the feature under `build/`, because the mutator rewrites the manifest in the file it tests on every run.
