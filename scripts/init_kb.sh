#!/bin/bash

# SalesMate AI Knowledge Base Initialization Script
# This script initializes the sales knowledge base with sample data

set -e

echo "🚀 SalesMate AI Knowledge Base Initialization"
echo "=============================================="

# Configuration
DATA_DIR="${SALESMATE_DATA:-./data}"
KB_DIR="${DATA_DIR}/sales_kb"
VECTORS_DIR="${DATA_DIR}/vectors"

# Create directories
echo "📁 Creating directories..."
mkdir -p "${KB_DIR}"
mkdir -p "${VECTORS_DIR}"
mkdir -p "${DATA_DIR}/sessions"
mkdir -p "${DATA_DIR}/cron"

# Initialize sample products
echo "📦 Initializing sample products..."
cat > "${KB_DIR}/products.json" << 'EOF'
[
  {
    "id": "basic",
    "name": "Basic Plan",
    "description": "Our entry-level solution for small businesses",
    "features": [
      "Up to 5 users",
      "Basic analytics",
      "Email support",
      "Standard reporting",
      "1GB storage"
    ],
    "use_cases": [
      "Small team collaboration",
      "Basic project management",
      "Simple workflow automation"
    ],
    "price": 29.00,
    "category": "starter",
    "tags": ["affordable", "entry", "small-business"],
    "technical_specs": {
      "Deployment": "Cloud",
      "API Access": "Limited",
      "SSO": "No",
      "Support": "Email"
    }
  },
  {
    "id": "pro",
    "name": "Professional Plan",
    "description": "Advanced features for growing businesses",
    "features": [
      "Up to 25 users",
      "Advanced analytics",
      "Priority support",
      "Custom reporting",
      "10GB storage",
      "API access"
    ],
    "use_cases": [
      "Mid-market teams",
      "Advanced analytics needs",
      "Custom integrations"
    ],
    "price": 79.00,
    "category": "professional",
    "tags": ["scalable", "mid-market", "advanced"],
    "technical_specs": {
      "Deployment": "Cloud",
      "API Access": "Full",
      "SSO": "Yes",
      "Support": "Priority"
    }
  },
  {
    "id": "enterprise",
    "name": "Enterprise Plan",
    "description": "Fully customizable solution for large organizations",
    "features": [
      "Unlimited users",
      "Advanced security",
      "Dedicated support",
      "Custom development",
      "Unlimited storage",
      "Premium API",
      "Single sign-on (SSO)",
      "Advanced reporting"
    ],
    "use_cases": [
      "Large enterprise",
      "Regulated industries",
      "Custom implementations"
    ],
    "price": 199.00,
    "category": "enterprise",
    "tags": ["enterprise", "secure", "customizable"],
    "technical_specs": {
      "Deployment": "Cloud/On-premise",
      "API Access": "Premium",
      "SSO": "Yes",
      "Support": "Dedicated"
    }
  }
]
EOF

# Initialize pricing tiers
echo "💰 Initializing pricing tiers..."
cat > "${KB_DIR}/pricing.json" << 'EOF'
[
  {
    "name": "Basic Monthly",
    "description": "Monthly billing for the Basic Plan",
    "price": 29.00,
    "features": {
      "Up to 5 users": true,
      "Basic analytics": true,
      "Email support": true,
      "Standard reporting": true,
      "1GB storage": true
    },
    "billing_cycle": "monthly",
    "min_quantity": 1
  },
  {
    "name": "Pro Monthly",
    "description": "Monthly billing for the Professional Plan",
    "price": 79.00,
    "features": {
      "Up to 25 users": true,
      "Advanced analytics": true,
      "Priority support": true,
      "Custom reporting": true,
      "10GB storage": true,
      "API access": true
    },
    "billing_cycle": "monthly",
    "min_quantity": 1
  },
  {
    "name": "Pro Annual",
    "description": "Annual billing for the Pro Plan (2 months free)",
    "price": 790.00,
    "features": {
      "Up to 25 users": true,
      "Advanced analytics": true,
      "Priority support": true,
      "Custom reporting": true,
      "10GB storage": true,
      "API access": true
    },
    "billing_cycle": "annual",
    "min_quantity": 1
  },
  {
    "name": "Enterprise Monthly",
    "description": "Monthly billing for the Enterprise Plan",
    "price": 199.00,
    "features": {
      "Unlimited users": true,
      "Advanced security": true,
      "Dedicated support": true,
      "Custom development": true,
      "Unlimited storage": true
    },
    "billing_cycle": "monthly",
    "min_quantity": 1
  }
]
EOF

# Initialize FAQ
echo "❓ Initializing FAQ..."
cat > "${KB_DIR}/faq.json" << 'EOF'
{
  "pricing": "We offer three pricing tiers: Basic ($29/month), Professional ($79/month), and Enterprise ($199/month). Each tier includes increasing levels of features and support. We also offer annual billing discounts (2 months free).",
  "free_trial": "Yes, we offer a 14-day free trial for all our plans. No credit card required to start your trial. You'll have full access to all features during the trial period.",
  "setup_time": "Most customers are up and running within 24 hours. Our onboarding team will guide you through the setup process and help customize the solution for your needs. Enterprise customers get dedicated onboarding support.",
  "integration": "We support integrations with popular tools including Salesforce, Slack, Microsoft Teams, Google Workspace, HubSpot, and many others. Our REST API allows for custom integrations with any system.",
  "support": "All plans include email support with 24-hour response time. Professional plans include priority support with 4-hour response time. Enterprise plans include dedicated account management and 1-hour response time.",
  "data_security": "We take data security seriously. All data is encrypted in transit (TLS 1.3) and at rest (AES-256). We comply with GDPR, CCPA, SOC 2 Type II, and ISO 27001 standards. Enterprise plans include additional security features.",
  "deployment": "Basic and Professional plans are cloud-hosted on AWS with multi-region support. Enterprise plans offer flexible deployment options including on-premise, private cloud, or hybrid configurations.",
  "on-premise": "Yes, we support on-premise deployment for Enterprise customers. This includes air-gapped environments for highly regulated industries. Contact our sales team for more information.",
  "api": "We provide a comprehensive REST API with full documentation. Professional and Enterprise plans include full API access. Rate limits vary by plan: Basic (1000 req/day), Pro (10000 req/day), Enterprise (unlimited).",
  "upgrade": "You can upgrade your plan at any time from the dashboard. The new features will be available immediately, and billing will be prorated. Downgrades take effect at the start of the next billing cycle."
}
EOF

# Initialize competitors info
echo "🏆 Initializing competitor information..."
cat > "${KB_DIR}/competitors.json" << 'EOF'
{
  "competitor_a": {
    "name": "Competitor A",
    "strengths": ["Market leader", "Strong brand recognition", "Extensive feature set"],
    "weaknesses": ["Higher price point", "Complex interface", "Steep learning curve", "Slow support"],
    "differentiators": ["Better user experience", "More affordable", "Easier to use", "Superior customer support", "Faster implementation"],
    "pricing_comparison": {
      "our_price": 79.00,
      "competitor_price": 129.00,
      "savings_percent": 39
    }
  },
  "competitor_b": {
    "name": "Competitor B",
    "strengths": ["Open source", "Highly customizable", "Active community", "Free to use"],
    "weaknesses": ["Requires technical expertise", "Limited out-of-box features", "Support challenges", "Security concerns"],
    "differentiators": ["Ready-to-use solution", "Professional support", "Continuous updates", "No maintenance overhead", "Enterprise-grade security"],
    "pricing_comparison": {
      "value_proposition": "We provide a complete, maintained solution with professional support",
      "total_cost_of_ownership": "Lower when factoring in internal resources needed for open-source solution"
    }
  }
}
EOF

# Initialize sales scripts
echo "📝 Initializing sales scripts..."
cat > "${KB_DIR}/scripts.json" << 'EOF'
{
  "greeting": "Hello! I'm SalesMate AI, your dedicated sales assistant. I'm here to help you find the perfect solution for your needs. How can I assist you today?",
  "discovery_questions": [
    "What specific challenges are you looking to solve?",
    "How large is your team that would be using this solution?",
    "What tools are you currently using for this?",
    "What's your timeline for implementing a new solution?"
  ],
  "objection_handling": {
    "price": "I understand budget is an important consideration. Let me show you the ROI our customers typically see - on average, they save 10+ hours per week and see a 3x return on their investment within the first quarter.",
    "competitor": "That's a great question. While [competitor] offers some similar features, our customers consistently tell us they prefer our solution because of our superior user experience, faster implementation, and dedicated support.",
    "timing": "I completely understand. Many of our most successful customers started with a free trial to see the value firsthand. Would you be interested in a 14-day trial with full access?"
  },
  "closing": "Based on our conversation, it sounds like our [Plan Name] would be a great fit for your needs. Would you like me to help you get started with a free trial, or do you have any other questions?"
}
EOF

# Create sample documents for RAG
echo "📚 Creating sample documents for RAG..."
mkdir -p "${KB_DIR}/documents"
cat > "${KB_DIR}/documents/product_overview.md" << 'EOF'
# SalesMate AI Product Overview

## What is SalesMate AI?

SalesMate AI is a 24/7 autonomous sales agent system that transforms how businesses handle sales conversations. It's not just a chatbot - it's an AI Sales Champion with the mindset of a top sales professional.

## Key Features

### Intelligent Conversation
- Natural language understanding with context awareness
- Sentiment analysis and emotion detection
- Multi-turn conversation handling
- Personalized responses based on customer profile

### Sales Methodology Integration
- SPIN Selling framework (Situation, Problem, Implication, Need-payoff)
- FAB selling (Features, Advantages, Benefits)
- BANT qualification (Budget, Authority, Need, Timeline)

### Multi-Channel Support
- Feishu (飞书)
- DingTalk (钉钉)
- WeCom (企业微信)
- Telegram
- Discord
- Slack
- WhatsApp
- Email

### Safety & Compliance
- Guardrails to prevent inappropriate commitments
- Emotion fuse for customer frustration detection
- Automatic human escalation when needed
- Audit logging for compliance

## Pricing

| Plan | Price | Users | Storage | Support |
|------|-------|-------|---------|---------|
| Basic | $29/mo | 5 | 1GB | Email |
| Professional | $79/mo | 25 | 10GB | Priority |
| Enterprise | $199/mo | Unlimited | Unlimited | Dedicated |

## Security & Compliance

- SOC 2 Type II certified
- GDPR compliant
- CCPA compliant
- ISO 27001 certified
- End-to-end encryption
- Data residency options
EOF

cat > "${KB_DIR}/documents/implementation_guide.md" << 'EOF'
# Implementation Guide

## Getting Started

### Prerequisites
- Docker and Docker Compose installed
- API keys for your preferred LLM provider (OpenAI, Anthropic, etc.)
- Channel credentials (Feishu, DingTalk, etc.)

### Quick Start

1. Clone the repository
2. Copy `.env.example` to `.env`
3. Fill in your API keys and credentials
4. Run `make docker-up`
5. Access the dashboard at http://localhost:18790

### Configuration

The system can be configured through:
- Environment variables
- Configuration files
- Dashboard settings

### Integration Steps

1. **Configure Channels**: Set up your messaging channels in the .env file
2. **Import Knowledge Base**: Add your product information, pricing, and FAQs
3. **Set Guardrails**: Configure safety limits and escalation rules
4. **Test**: Use the test mode to verify responses
5. **Deploy**: Switch to production mode

### Best Practices

1. Start with the shadow mode (AI suggests, human approves)
2. Review conversation logs regularly
3. Update knowledge base with new information
4. Monitor conversion metrics
5. Fine-tune based on feedback

### Support

For implementation support:
- Email: support@salesmate.ai
- Documentation: docs.salesmate.ai
- Community: community.salesmate.ai
EOF

echo "✅ Knowledge base initialization complete!"
echo ""
echo "Directory structure:"
echo "  ${KB_DIR}/products.json   - Product catalog"
echo "  ${KB_DIR}/pricing.json    - Pricing information"
echo "  ${KB_DIR}/faq.json        - Frequently asked questions"
echo "  ${KB_DIR}/competitors.json - Competitor information"
echo "  ${KB_DIR}/scripts.json    - Sales scripts"
echo "  ${KB_DIR}/documents/      - RAG documents"
echo ""
echo "Next steps:"
echo "  1. Edit .env with your API keys"
echo "  2. Run 'make docker-up' to start services"
echo "  3. Access the gateway at http://localhost:18790"