# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SalesMate AI is a 24/7 autonomous sales agent system built as an evolution of the nanotalon personal AI assistant. The system transforms a general AI assistant into a specialized "AI Sales Champion" (AI 销冠伙伴) that can independently handle sales processes from initial contact through to closing deals.

## Architecture

### Core Components
- `agent/`: Base agent functionality including loop, context, memory, skills, and tools
- `sales_agent/`: Sales-specific extensions including sales loop, knowledge base, pipeline manager, and emotion analyzer
- `sales_intelligence/`: Sales strategy engines implementing SPIN, FAB, and BANT methodologies
- `security/`: Guardrails system to prevent inappropriate sales behaviors
- `knowledge/`: Sales knowledge base with RAG (Retrieval-Augmented Generation) capabilities
- `channels/`: Multi-platform messaging integration (Feishu, DingTalk, Telegram, etc.)
- `providers/`: LLM provider integrations
- `config/`: Configuration management

### Key Modules
- `sales_agent/sales_loop.go`: Extends the base AgentLoop with sales-specific functionality, stage management, and emotional analysis
- `sales_intelligence/strategy_engine.go`: Implements SPIN, FAB, and BANT sales methodologies
- `sales_agent/pipeline_manager.go`: Manages the sales pipeline and lead progression
- `security/guardrails.go`: Implements safety mechanisms to prevent inappropriate sales commitments
- `knowledge/sales_kb.go`: Sales-specific knowledge base with product, pricing, and FAQ management

## Development Commands

### Building
```bash
# Build the entire project
go build ./...

# Build specific modules
go build ./sales_agent/...
go build ./sales_intelligence/...
go build ./security/...
go build ./knowledge/...
```

### Running
```bash
# Initialize the system (after configuration)
go run main.go

# Run using the CLI commands
./salesmate agent -m "Hello, I'm interested in your product"
./salesmate gateway  # Starts the full service
./salesmate channels status
./salesmate status
```

### Testing
```bash
# Since no dedicated test files exist yet, test by building individual packages
go build ./sales_agent/...
go build ./sales_intelligence/...

# Or run the test binary created during development
./test_salesmate
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Vendor dependencies (optional)
go mod vendor
```

## Key Design Patterns

### Agent Extension Pattern
The system extends the base AgentLoop with sales-specific functionality through composition rather than inheritance. The SalesLoop embeds base functionality while adding sales-specific capabilities.

### Pipeline State Management
Sales stages follow a progression: NewContact → Discovery → Presentation → Negotiation → Close, with automatic transitions based on customer intent and engagement metrics.

### Security Through Guardrails
Critical sales functions like pricing, contracts, and feature claims are protected by configurable guardrails that can soft-alert, hard-block, or require review.

### Intent Recognition
Customer messages are analyzed for sales-specific intents (pricing, demos, objections, etc.) to drive appropriate responses and pipeline progression.

## Common Development Tasks

### Adding New Sales Features
1. Extend the `SalesKnowledgeBase` with new product information
2. Update intent recognizers in `sales_intelligence` if new intent categories are needed
3. Modify stage transition logic in the `SalesLoop` if necessary
4. Add appropriate guardrails in the security module

### Configuring Multi-Channel Support
1. Update channel configurations in `channels/`
2. Ensure proper context switching for sales sessions across platforms
3. Test guardrail enforcement on each platform

### Extending Sales Methodologies
1. Add new strategy patterns in `sales_intelligence/strategy_engine.go`
2. Integrate with the existing SPIN, FAB, BANT frameworks
3. Update the sales loop to utilize new strategies based on context

## Important Notes

- The system has been migrated from the original "nanotalon" module to "salesmate" with all import paths updated
- The codebase maintains compatibility with the original multi-channel messaging architecture while adding sales-specific features
- Guardrails are essential for preventing inappropriate sales behaviors and must be carefully configured
- The sales pipeline management system maintains persistent state for customer progression through sales stages
- RAG (Retrieval-Augmented Generation) is used to ensure factually accurate responses based on company knowledge