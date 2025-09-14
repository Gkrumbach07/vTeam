# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**vTeam** is a comprehensive AI automation platform containing:

1. **RAT (Refinement Agent Team) System**: An AI-powered automation system to reduce engineering refinement time and improve ticket quality
2. **Ambient Agentic Runner**: A Kubernetes-native platform for running automated agentic sessions with AI and MCP capabilities
3. **vTeam Tools**: Supporting tools including shared configurations and MCP client integration

## Architecture

### RAT System (`demos/rfe-builder/`)
- **Technology Stack**: Streamlit web application with Python backend
- **AI Integration**: Anthropic Claude API with Vertex AI support for conversational RFE creation
- **Data Models**: Pydantic models for RFE workflow management (7-step council review process)
- **Agent System**: Multi-agent workflow with specialized roles (PM, Architect, Staff Engineer, PO, Team Lead, Team Member, Delivery Owner)

### Ambient Agentic Runner (`components/`)
- **Technology Stack**: Kubernetes-native with Go backend, NextJS frontend, Python AI service
- **AI Integration**: Ambient Code AI with MCP server capabilities for browser automation
- **Architecture**: Microservices with Custom Resources, Operators, and Job execution
- **Multi-tenancy**: Namespace-based isolation with RBAC integration and policy enforcement
- **Capabilities**: Generic agentic task execution including website analysis, automation, and data processing
- **Session CRD**: Generic session management supporting multiple agentic frameworks
- **Webhook Integration**: External tool integration (GitHub, Jira, Slack) with API key authentication

### vTeam Tools (`tools/`)
- **vTeam Shared Configs**: CLI tool (`vteam-config`) for Claude Code configuration management
- **MCP Client Integration**: Python library for Model Context Protocol client integration
- **Installation**: Individual tools with separate setup and maintenance

## Development Commands

### Python Environment Setup
```bash
# Use virtual environments for all Python development
python -m venv venv
source venv/bin/activate  # or `venv\Scripts\activate` on Windows

# Install dependencies (prefer uv over pip)
uv pip install -r requirements.txt
```

### RFE Builder Demo
```bash
cd demos/rfe-builder
streamlit run app.py
```

### Tool Development (vTeam Tools)
```bash
# vTeam Shared Configs
cd tools/vteam_shared_configs
uv pip install -e ".[dev]"
black .
isort --profile black .
flake8 --max-line-length=88 --extend-ignore=E203,W503 .
python -m pytest

# MCP Client Integration
cd tools/mcp_client_integration
uv pip install -e .
python -m pytest tests/
```

### Pre-commit Hooks
```bash
# Install pre-commit hooks (configured in demos/rfe-builder/.pre-commit-config.yaml)
pre-commit install
pre-commit run --all-files
```

## Key Architecture Patterns

### RAT System Workflow
1. **7-Step Council Process**: RFEs flow through specialized agent roles
2. **State Management**: WorkflowState class manages RFE lifecycle with Pydantic models
3. **Agent Roles**: Enum-based agent system with predefined responsibilities
4. **Conversational AI**: ChatInterface provides natural language RFE creation
5. **Cost Tracking**: Built-in usage monitoring for AI API calls

### Ambient Platform Architecture
1. **Multi-tenant Sessions**: Generic Session CRD with namespace-scoped isolation
2. **External Tool Integration**: Webhook processing with API key authentication
3. **Policy Enforcement**: Namespace policies constraining models, tools, budgets
4. **RBAC Integration**: Kubernetes-native access control for viewer/editor permissions
5. **Framework Agnostic**: Pluggable agentic framework support via generic artifact references
6. **Audit Trail**: Append-only history for compliance and debugging

### Agent Role Mapping
- **Parker (PM)**: Prioritizes RFEs and communicates assessments
- **Archie (Architect)**: Reviews RFEs and checks acceptance criteria  
- **Stella (Staff Engineer)**: Performs completeness checks and accept/reject decisions
- **Olivia (PO)**: Manages acceptance criteria
- **Lee (Team Lead)**: Coordinates team execution
- **Taylor (Team Member)**: Handles implementation details
- **Derek (Delivery Owner)**: Creates feature tickets and manages delivery

### Configuration Management
- **Symlink-based**: Global configs linked to user's `.claude/` directory
- **Template System**: Project templates for different development scenarios
- **Team Hooks**: Shared Git hooks and development standards
- **Local Overrides**: `.claude/settings.local.json` for personal customization

## Testing Strategy
- **Unit Tests**: pytest framework with coverage reporting
- **Type Checking**: mypy with additional dependencies for Pydantic
- **Linting**: black, isort, flake8 with pre-commit automation
- **AI Integration Tests**: Mock Anthropic API calls for testing

## File Structure Significance

```
vTeam/
├── components/                   # Ambient Agentic Runner services  
│   ├── frontend/                 # NextJS web interface
│   ├── backend/                  # Go API service
│   ├── operator/                 # Kubernetes operator
│   ├── runners/                 # AI runner services
│   │   └── claude-code-runner/  # Python Claude Code CLI service
│   └── manifests/                # Kubernetes deployment files
├── tools/                        # Supporting tools and utilities
│   ├── vteam_shared_configs/     # Team configuration package
│   │   ├── cli.py                # Command-line interface (vteam-config)
│   │   ├── installer.py          # Configuration management logic
│   │   └── pyproject.toml        # Package configuration
│   └── mcp_client_integration/   # MCP client library
├── demos/rfe-builder/            # RAT system demonstration
│   ├── app.py                    # Main Streamlit application
│   └── src/                      # Demo source code
└── rhoai-ux-agents-vTeam.md     # Complete agent framework documentation
```

## AI Integration Notes
- **Primary API**: Anthropic Claude with Vertex AI fallback
- **Model Selection**: Configurable via `get_model_name()` function
- **Cost Management**: Built-in token counting and caching system
- **Error Handling**: Graceful degradation when AI services unavailable

## Agent-Driven Development
This project implements an AI agent system modeled after real software teams with different seniority levels and responsibilities. The agent framework (detailed in `rhoai-ux-agents-vTeam.md`) provides realistic team dynamics for AI-assisted development workflows.

## Configuration Standards
- **Python Formatting**: black (88 char line length, double quotes)
- **Import Sorting**: isort with black profile
- **Linting**: flake8 with line length 88, ignoring E203,W503
- **Type Checking**: mypy with missing import tolerance
- **Git Workflow**: Feature branches, squashed commits, conventional commit messages