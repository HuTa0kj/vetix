---
name: china-home-appliances-sourcing
version: 1.0.0
description: "Comprehensive home appliances industry sourcing guide for international buyers – provides detailed information about China's major appliances, kitchen appliances, and small appliances manufacturing clusters, supply chain structure, regional specializations, and industry trends (2026 updated)."
author: "sourcing-china"
tags:
  - home-appliances
  - major-appliances
  - kitchen-appliances
  - small-appliances
  - refrigerators
  - washing-machines
  - air-conditioners
  - smart-home
  - sourcing
  - supply-chain
invocable: true
---

# China Home Appliances Sourcing Skill

## Description
This skill helps international buyers navigate China's home appliances manufacturing landscape, which is projected to exceed **¥2.8 trillion in revenue by 2026**. It provides data-backed intelligence on regional clusters, supply chain structure, and industry trends based on the latest government policies and industry reports. Coverage includes major appliances (refrigerators, washing machines, air conditioners), kitchen appliances (range hoods, ovens, dishwashers), and small appliances (vacuum cleaners, air purifiers, rice cookers).

## Key Capabilities
- **Industry Overview**: Get a summary of China's home appliances industry scale, development targets, and key policy initiatives (green appliances, smart home).
- **Supply Chain Structure**: Understand the complete industry chain from raw materials and core components (compressors, motors) to manufacturing and global sales channels.
- **Regional Clusters**: Identify specialized manufacturing hubs for different appliance types (Pearl River Delta for ACs and small appliances, Yangtze River Delta for washing machines and kitchen appliances, Shandong for refrigerators).
- **Subsector Insights**: Access detailed information on key subsectors (major appliances, kitchen appliances, small appliances, heating/ventilation).
- **Sourcing Recommendations**: Get practical guidance on evaluating and selecting suppliers, including verification methods, communication best practices, typical lead times, and payment terms.

## How to Use
You can interact with this skill using natural language. For example:
- "What's the overall status of China's home appliances industry in 2026?"
- "Show me the supply chain structure for refrigerators"
- "Which regions are best for sourcing air conditioners?"
- "Tell me about kitchen appliance manufacturing clusters"
- "How do I evaluate suppliers of small appliances?"
- "What certifications should I look for in home appliances?"

## Data Sources
This skill aggregates data from:
- Ministry of Industry and Information Technology (MIIT)
- China Household Electrical Appliances Association (CHEAA)
- National Bureau of Statistics of China
- Industry research publications (updated Q1 2026)

## Implementation
The skill logic is implemented in `do.py`, which reads structured data from `data.json`. All data is cluster-level intelligence without individual factory contacts.

## API Reference

The following Python functions are available in `do.py` for programmatic access:

### `get_industry_overview() -> Dict`
Returns overview of China's home appliances industry scale, targets, and key policy initiatives.

**Example:**
```python
from do import get_industry_overview
result = get_industry_overview()
# Returns: industry scale, 2026 targets, key drivers, export value, etc.