# AI Assistant Rules

@AI.md PART 0: AI ASSISTANT RULES

## Critical Behaviors
- NEVER create documentation files unless explicitly asked
- NEVER create AUDIT.md, REVIEW.md, TODO.md - FIX issues directly
- ALWAYS search before adding (avoid duplicates)
- ALWAYS verify against AI.md spec before implementing
- Container-only development (Docker/Incus)

## Reading AI.md
- File is ~1.7MB, ~46,000 lines - read PART by PART
- ALWAYS read PART 0 and PART 1 first
- Use grep to find relevant sections
- Re-read relevant PART before each task

## Never Guess or Assume
- Unsure? STOP and ASK
- Can't find file? Search first, then ask
- Multiple approaches? List options, ask user
- Spec incomplete? Ask for clarification

## Speed vs Correctness
1. Correct (first priority)
2. Verified (second priority)
3. Fast (last priority)

A fast wrong answer is WORSE than a slow correct answer.
