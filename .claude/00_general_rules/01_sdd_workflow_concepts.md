# Spec-Driven Development (SDD) Workflow

This document outlines the methodology used to understand, plan, and execute software development tasks within this project. The philosophy is inspired by spec-first approaches like that of AWS's Kiro, ensuring clarity, alignment, and robust documentation before implementation begins.

## 1. Philosophy and Goals

The primary goal of this workflow is to move from a user request to a well-understood and planned implementation. By creating specification documents *before* writing the bulk of the code, we achieve several key objectives:

*   **Clarity of Thought:** It forces a deep understanding of the problem and the existing codebase before making changes.
*   **User Alignment:** It provides a clear plan that can be reviewed and approved by the user, ensuring we are building the right thing.
*   **Structured Process:** It breaks down complex tasks into manageable, sequential steps.
*   **Documentation as a Byproduct:** The specification files serve as valuable, persistent documentation for the components they describe.

## 2. The Core Artifacts: The `.spec` Directory

For any significant task (new feature, refactoring, etc.), a `.spec` directory is created. This directory is co-located with the code it describes (e.g., `pkg/builder/.spec/` for the builder package, or a root `.spec/` for project-wide concerns).

This directory contains three key documents with strict separation of concerns:

1.  **`requirements.md`**: The **"What"** - Pure Requirements
    - Lists functional and non-functional requirements using EARS notation
    - Each requirement has simple PASS/FAIL status
    - Contains acceptance criteria for each requirement
    - **Does NOT contain**: Problem analysis, implementation details, or task planning

2.  **`design.md`**: The **"How"** - Technical Analysis & Architecture
    - Describes high-level architecture and design patterns
    - Contains problem analysis and technical investigation
    - Explains current issues and proposed solutions
    - Documents design decisions and trade-offs
    - **Does NOT contain**: Requirements definitions or implementation steps

3.  **`tasks.md`**: The **"When & Order"** - Pure Implementation Plan
    - Provides concrete, sequential implementation steps
    - Contains time estimates and dependencies
    - Lists "Done When" criteria for each task
    - **Does NOT contain**: Problem analysis, requirements, or architectural explanations

## 3. The Step-by-Step Workflow

1.  **Understand the Goal:** Receive and clarify the user's request.

2.  **Analyze the Codebase:** Use tools like `glob`, `read_file`, and `search_file_content` to thoroughly understand the existing code, its structure, and its conventions.

3.  **Prepare the Specification (`.spec` files):**
    *   **ALWAYS ensure** the `.spec` directory exists in the appropriate location for significant tasks.
    *   **If `.spec` files exist**: Read and understand them first, then update as needed.
    *   **If `.spec` files don't exist**: Create them from scratch.
    *   Ensure `requirements.md` details what the final code must be able to do.
    *   Ensure `design.md` explains how the code will be structured to meet those requirements.
    *   Ensure `tasks.md` provides a clear, step-by-step implementation plan.
    *   **Note**: Preparing `.spec` documentation is **encouraged and expected** for complex work - this is separate from the general prohibition on creating other documentation files.

4.  **Seek User Approval:** Present the plan (usually by showing the contents of `tasks.md` or summarizing the design) to the user for confirmation. **Do not proceed with major implementation changes without this alignment.**

5.  **Implement:** Execute the plan outlined in `tasks.md`. This involves writing, modifying, and deleting code and tests using the available tools.

6.  **Verify:** Run tests and any other checks to ensure the implementation is correct and fully meets the criteria defined in `requirements.md`.

By following this structured process, we ensure that all work is deliberate, well-planned, and aligned with the user's goals, resulting in a more robust and maintainable codebase.

## 3.1. Maintaining and Updating `.spec` Documents

**Ongoing Responsibility**: `.spec` documents are **living documentation** that must be kept current with code changes.

### When to Update `.spec` Files:

1. **During Code Modifications**: When touching any package with existing `.spec` files
2. **Architecture Changes**: When modifying interfaces, data flow, or design patterns
3. **Requirement Changes**: When user needs evolve or new constraints are discovered
4. **Bug Fixes**: When fixes reveal gaps in original requirements or design

### Update Process:

1. **Before Code Changes**: Review existing `.spec` files to understand current documented behavior
2. **During Implementation**: Update relevant sections as changes are made:
   - `requirements.md`: Add/modify requirements if scope changes
   - `design.md`: Update architecture diagrams and design decisions
   - `tasks.md`: Mark completed tasks, add new ones if scope expands
3. **After Code Changes**: Verify `.spec` files accurately reflect the final implementation

### Maintenance Triggers:

- **Package Analysis**: When analyzing packages (like `/analyze`), always check and update `.spec` files
- **Issue Discovery**: When finding problems (race conditions, TODOs), document them in appropriate `.spec` files
- **Feature Additions**: Extend existing `.spec` files rather than creating disconnected new ones
- **Refactoring**: Update design.md to reflect new patterns and architectural decisions

**Key Principle**: `.spec` files should always represent the **current state** and **planned evolution** of the codebase, not just historical decisions.

## 3.2. Separation of Concerns Guidelines

**CRITICAL**: Maintain strict boundaries between the three spec files to avoid overlap and confusion.

### What Goes Where

**requirements.md**:
✅ **Include**: REQ-001 style requirements, PASS/FAIL status, acceptance criteria  
❌ **Never Include**: "Current Issues", "TODO items", "Implementation Priority", technical analysis

**design.md**:
✅ **Include**: Architecture diagrams, technical problem analysis, proposed solutions, current issues  
❌ **Never Include**: Requirements definitions, implementation steps, task schedules

**tasks.md**:
✅ **Include**: TASK-001 style steps, time estimates, "Done When" criteria, implementation order  
❌ **Never Include**: Problem analysis, requirements, architectural explanations

### Common Anti-Patterns to Avoid

❌ **Overlap**: Repeating the same information across multiple files  
❌ **Task-like Requirements**: "Must fix race conditions immediately" (belongs in tasks.md)  
❌ **Requirements in Design**: "REQ-30: Thread Safety" (belongs in requirements.md)  
❌ **Analysis in Tasks**: "Race conditions occur because..." (belongs in design.md)

### File Reference Pattern

Each file should reference others appropriately:
- **tasks.md**: "See design.md for technical analysis and requirements.md for acceptance criteria"
- **design.md**: "Addresses REQ-30 from requirements.md"  
- **requirements.md**: "Implementation tracked in tasks.md"

## 4. EARS Notation for Requirements

The `requirements.md` file should use EARS (Easy Approach to Requirements Syntax) notation to ensure requirements are clear, testable, and unambiguous. EARS provides five template patterns for different types of requirements.

### 4.1 EARS Template Patterns

#### Ubiquitous Requirements (Always Active)
For requirements that apply at all times:
```
REQ-[ID]: The <system/component> SHALL <system response>
```

**Example:**
```
REQ-001: The parser SHALL validate all input files before processing
REQ-002: The emitter SHALL generate syntactically valid Go code
```

#### Event-Driven Requirements
For requirements triggered by specific events:
```
REQ-[ID]: WHEN <trigger> the <system/component> SHALL <system response>
```

**Example:**
```
REQ-010: WHEN a `:skip` annotation is encountered the parser SHALL exclude the specified field from mapping
REQ-011: WHEN validation fails the system SHALL return a descriptive error message
```

#### State-Driven Requirements
For requirements that depend on system state:
```
REQ-[ID]: WHILE <in a specific state> the <system/component> SHALL <system response>
```

**Example:**
```
REQ-020: WHILE processing type annotations the builder SHALL maintain field mapping context
REQ-021: WHILE in error state the coordinator SHALL halt pipeline execution
```

#### Optional Feature Requirements
For conditional functionality:
```
REQ-[ID]: WHERE <feature is included> the <system/component> SHALL <system response>
```

**Example:**
```
REQ-030: WHERE optimization is enabled the emitter SHALL apply dead code elimination
REQ-031: WHERE debug mode is active the system SHALL log detailed execution steps
```

#### Complex Requirements (Multiple Conditions)
For requirements with multiple triggers or conditions:
```
REQ-[ID]: WHEN <trigger> AND WHERE <condition> the <system/component> SHALL <system response>
```

**Example:**
```
REQ-040: WHEN a conversion error occurs AND WHERE retry is enabled the executor SHALL attempt the operation up to 3 times
REQ-041: WHEN type casting is requested AND WHERE types are compatible the generator SHALL emit appropriate cast syntax
```

### 4.2 Requirements Structure Guidelines

#### Requirement Categories
Organize requirements into clear categories:

- **Functional Requirements (FR-xxx)**: What the system must do
- **Non-Functional Requirements (NFR-xxx)**: Quality attributes (performance, reliability, etc.)
- **Interface Requirements (IR-xxx)**: External interfaces and APIs
- **Constraint Requirements (CR-xxx)**: Design and implementation constraints

#### Requirement Attributes
Each requirement should include:

```markdown
**REQ-001: System Validation**
- **Type**: Functional
- **Priority**: Must Have
- **Description**: The parser SHALL validate all input files before processing
- **Rationale**: Prevents runtime errors and ensures data integrity
- **Acceptance Criteria**: 
  - Invalid Go files are rejected with clear error messages
  - Validation occurs before any processing begins
  - All syntax errors are detected and reported
- **Dependencies**: None
- **Verification Method**: Unit tests with invalid input files
```

#### Priority Levels
Use MoSCoW prioritization:
- **Must Have**: Critical functionality
- **Should Have**: Important but not critical
- **Could Have**: Nice to have features
- **Won't Have**: Explicitly excluded features

### 4.3 Example Requirements Document Structure

```markdown
# Requirements for [Component Name]

## Overview
Brief description of the component and its purpose.

## Functional Requirements

### FR-001: Core Processing
**Type**: Functional  
**Priority**: Must Have  
**Description**: The parser SHALL extract interface definitions from Go source files  
**Rationale**: Core functionality required for code generation  
**Acceptance Criteria**:
- Interfaces marked with `:convergen` are identified
- Method signatures are correctly parsed
- Annotation comments are extracted and associated with methods
**Verification Method**: Integration tests with sample Go files

### FR-002: Error Handling
**Type**: Functional  
**Priority**: Must Have  
**Description**: WHEN invalid syntax is encountered the parser SHALL return descriptive error messages  
**Rationale**: Enables users to quickly identify and fix issues  
**Acceptance Criteria**:
- Error messages include line numbers and file names
- Multiple errors are collected and reported together
- Error messages are human-readable and actionable
**Verification Method**: Unit tests with malformed input

## Non-Functional Requirements

### NFR-001: Performance
**Type**: Non-Functional  
**Priority**: Should Have  
**Description**: The parser SHALL process files with less than 500ms latency for files under 1MB  
**Rationale**: Ensures good developer experience during code generation  
**Acceptance Criteria**:
- 95th percentile response time < 500ms for 1MB files
- Memory usage remains below 100MB during processing
- Processing time scales linearly with file size
**Verification Method**: Performance benchmarks

## Interface Requirements

### IR-001: CLI Interface
**Type**: Interface  
**Priority**: Must Have  
**Description**: The system SHALL provide a command-line interface accepting file paths as arguments  
**Rationale**: Standard interface for Go code generation tools  
**Acceptance Criteria**:
- Accepts input file path as positional argument
- Supports standard flags (--help, --version, --output)
- Returns appropriate exit codes (0 for success, non-zero for errors)
**Verification Method**: CLI integration tests

## Constraint Requirements

### CR-001: Go Compatibility
**Type**: Constraint  
**Priority**: Must Have  
**Description**: The generated code SHALL be compatible with Go 1.21+  
**Rationale**: Ensures compatibility with modern Go toolchain  
**Acceptance Criteria**:
- Generated code compiles without errors on Go 1.21+
- Uses only language features available in target Go version
- Imports are properly formatted and valid
**Verification Method**: Automated testing across Go versions
```

### 4.4 Best Practices for EARS Requirements

1. **Use Active Voice**: Write requirements in active voice for clarity
2. **Be Specific**: Avoid vague terms like "user-friendly" or "efficient"
3. **Make Testable**: Each requirement should be verifiable through testing
4. **Avoid Implementation Details**: Focus on what, not how
5. **Use Consistent Terminology**: Maintain a glossary of terms
6. **Number Systematically**: Use consistent numbering scheme (REQ-001, REQ-002, etc.)
7. **Cross-Reference**: Link related requirements and trace to design decisions

This structured approach ensures that requirements are clear, testable, and provide a solid foundation for design and implementation.
