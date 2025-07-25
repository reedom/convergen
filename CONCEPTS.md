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

This directory contains three key documents:

1.  **`requirements.md`**: The **"What"**. This file lists the functional and non-functional requirements for the component. Each requirement should be a clear, verifiable statement (e.g., "REQ-1: The parser MUST handle `:skip` notations.").

2.  **`design.md`**: The **"How"**. This file describes the high-level architecture and design of the component. It should explain the key structs, interfaces, data flow, and design patterns used to meet the requirements.

3.  **`tasks.md`**: The **"Plan"**. This file is a checklist of the concrete, sequential steps that will be taken to implement the design. It translates the abstract design into an actionable implementation plan.

## 3. The Step-by-Step Workflow

1.  **Understand the Goal:** Receive and clarify the user's request.

2.  **Analyze the Codebase:** Use tools like `glob`, `read_file`, and `search_file_content` to thoroughly understand the existing code, its structure, and its conventions.

3.  **Create the Specification (`.spec` files):**
    *   Create the `.spec` directory in the appropriate location.
    *   Write `requirements.md`, detailing what the final code must be able to do.
    *   Write `design.md`, explaining how the code will be structured to meet those requirements.
    *   Write `tasks.md`, providing a clear, step-by-step implementation plan.

4.  **Seek User Approval:** Present the plan (usually by showing the contents of `tasks.md` or summarizing the design) to the user for confirmation. **Do not proceed with major changes without this alignment.**

5.  **Implement:** Execute the plan outlined in `tasks.md`. This involves writing, modifying, and deleting code and tests using the available tools.

6.  **Verify:** Run tests and any other checks to ensure the implementation is correct and fully meets the criteria defined in `requirements.md`.

By following this structured process, we ensure that all work is deliberate, well-planned, and aligned with the user's goals, resulting in a more robust and maintainable codebase.
