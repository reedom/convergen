# Spec-Driven Development (SDD) Workflow

This document outlines the methodology for understanding, planning, and executing complex software development tasks. The approach ensures clarity, alignment, and proper documentation before implementation.

## 1. Philosophy and Goals

The SDD workflow achieves structured development through specification-first planning:

- **Clarity**: Deep understanding of problems before coding
- **Alignment**: Clear plans reviewed by users before implementation
- **Structure**: Complex tasks broken into manageable steps
- **Documentation**: Specs serve as living project documentation

## 2. The `.spec` Directory Structure

For significant tasks (new features, refactoring, architectural changes), create a `.spec` directory co-located with the code:
- Package-specific: `pkg/builder/.spec/`
- Project-wide: `.spec/` (root level)

### Three Core Documents

**`requirements.md`** - **"What"** (Pure Requirements)
- Functional and non-functional requirements using EARS notation
- **Include**: REQ-001 style requirements, PASS/FAIL status, Implementation Priority, acceptance criteria
- **Never Include**: "Current Issues", "TODO items", technical analysis

**`design.md`** - **"How"** (Technical Analysis & Architecture)
- Architecture diagrams and design patterns
- Problem analysis and proposed solutions
- Design decisions and trade-offs
- **Include**: Architecture diagrams, technical problem analysis, proposed solutions, current issues
- **Never Include**: Requirements definitions, implementation steps, task schedules

**`tasks.md`** - **"When & Order"** (Implementation Plan)
- Sequential implementation steps with clear completion criteria
- Progress tracking with checkboxes
- **Include**: TASK-001 style numbered tasks, requirement codes (FR-001, NFR-002), checkbox progress tracking `[ ]`
- **Never Include**: Time estimates, problem analysis, requirements definitions, architectural explanations, verbose descriptions, methodology discussions

### Separation of Concerns Guidelines

**CRITICAL**: Maintain strict boundaries between the three spec files to avoid overlap and confusion.

#### Common Anti-Patterns to Avoid

❌ **Overlap**: Repeating the same information across multiple files
❌ **Task-like Requirements**: "Must fix race conditions immediately" (belongs in tasks.md)
❌ **Requirements in Design**: "REQ-30: Thread Safety" (belongs in requirements.md)
❌ **Analysis in Tasks**: "Race conditions occur because..." (belongs in design.md)

### File Reference Pattern

Each file should reference others appropriately:
- **tasks.md**: "See design.md for technical analysis and requirements.md for acceptance criteria"
- **design.md**: "Addresses REQ-30 from requirements.md"
- **requirements.md**: "Implementation tracked in tasks.md"

## 3. Workflow Steps

### Discovery & Planning
1. **Find existing docs**: `find /project -type d -name ".spec*"`
2. **Update vs Create**: Update existing .spec/{requirements,design,tasks}.md files OR create new ones
3. **Analyze codebase**: Understand structure and conventions
4. **Get approval**: User confirms plan before major changes

### Implementation
1. **Execute**: Follow tasks.md plan
2. **Verify**: Run tests and linting
3. **Update**: Keep .spec/{requirements,design,tasks}.md files current with changes

## 4. EARS Notation for Requirements

Use EARS (Easy Approach to Requirements Syntax) for clear, testable requirements:

### Basic EARS Patterns

**Always Active:**
```
REQ-[ID]: The <system> SHALL <response>
```

**Event-Driven:**
```
REQ-[ID]: WHEN <trigger> the <system> SHALL <response>
```

**State-Driven:**
```
REQ-[ID]: WHILE <state> the <system> SHALL <response>
```

**Conditional:**
```
REQ-[ID]: WHERE <condition> the <system> SHALL <response>
```

### Example Requirements Structure

```markdown
## Functional Requirements

### REQ-001: Input Validation
**Priority**: Must Have
**Description**: The parser SHALL validate input files before processing
**Acceptance Criteria**:
- Invalid files rejected with clear errors
- Validation occurs before any processing

### REQ-002: Error Reporting
**Priority**: Must Have
**Description**: WHEN validation fails the system SHALL return descriptive error messages
**Acceptance Criteria**:
- Error messages include line numbers and file names
- Multiple errors collected and reported together
```

### Categories & Priorities

**Categories:**
- **FR-xxx**: Functional Requirements
- **NFR-xxx**: Non-Functional Requirements (performance, security)
- **CR-xxx**: Constraint Requirements (compatibility, standards)

**Priorities:** Must Have, Should Have, Could Have, Won't Have
