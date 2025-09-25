# Systems Thinking API - Updated Documentation Summary

## API Overview

This document summarizes the updated API documentation for the systems thinking backend after cleanup and streamlining.

## Key Changes Made

1. **Removed Redundant Endpoints**: Eliminated pseudo-functions and unnecessary CRUD operations that duplicated functionality
2. **Essential-Only API**: Maintained only the 13 essential handlers specified by user requirements
3. **Request/Response Model Separation**: Updated documentation to reflect proper Request models for input and Response models for output
4. **Consistent ID Format**: Documented the `type_timestamp` ID format used throughout the system

## Current API Structure (13 Essential Endpoints)

### Narrative Management (4 endpoints)
- `POST /narratives` - Create narrative
- `GET /narratives/{id}` - Get narrative by ID
- `PUT /narratives/{id}` - Update narrative
- `DELETE /narratives/{id}` - Delete narrative

### Node Creation (3 endpoints)
- `POST /nodes/system` - Create system node
- `POST /nodes/stock` - Create stock node (with type validation: accumulation|buffer|delay)
- `POST /nodes/flow` - Create flow node (with type validation: inflow|outflow|connector)

### Relationship Creation (5 endpoints)
- `POST /relationships/describes` - Link narrative to system
- `POST /relationships/describes-static` - Link stock to system
- `POST /relationships/describes-dynamic` - Link flow to system
- `POST /relationships/constitutes` - Create system hierarchy
- `POST /relationships/changes` - Link flow to stock with polarity validation

### Causal Link Creation (1 endpoint)
- `POST /causal-link` - Create causal relationships between nodes

## Updated Documentation Files

### 1. systems_thinking_api_endpoints.csv
- **Purpose**: Complete API endpoint reference for prompt workflow integration
- **Contents**: Method, endpoint path, request/response types, examples, validation notes
- **Key Features**: 
  - Updated example IDs using current timestamp format
  - Proper Request model naming (e.g., CreateNarrativeRequest)
  - Validation requirements clearly documented (Stock.Type, Changes.Polarity)

### 2. systems_thinking_data_models.csv
- **Purpose**: Data model reference with all Request and Response types
- **Contents**: Field-by-field documentation for all models used by the API
- **Key Features**:
  - Separate documentation for Request models (input) and Response models (output)
  - Validation requirements and constraints clearly specified
  - JSON behavior documented (always present, omitempty, etc.)

## ID Generation Format

All entities use auto-generated IDs with the format: `{type}_{timestamp}`

Examples:
- `narrative_1758811810649358883`
- `system_1758811810649358884` 
- `stock_1758811810649358885`
- `flow_1758811810649358886`

## Validation Rules

### Stock Type Validation
- Must be: `accumulation`, `buffer`, or `delay`
- Required field, cannot be empty

### Flow Type Validation  
- Must be: `inflow`, `outflow`, or `connector`
- Required field, cannot be empty

### Changes Polarity Validation
- Must be: `1.0` (positive influence) or `-1.0` (negative influence)
- Used in flow-stock relationships

## Integration Notes

This cleaned API is designed for integration with prompt workflows that generate function calls. The documentation provides:

1. **Clear Request Formats**: Exact JSON structure needed for each endpoint
2. **Response Examples**: What to expect back from each call
3. **Validation Requirements**: What fields are required and their constraints
4. **Relationship Modeling**: How to link entities together in the graph database

## Benefits of Cleanup

1. **Reduced Complexity**: From 30+ endpoints down to 13 essential ones
2. **Better Maintainability**: Clearer separation between request/response models
3. **Improved Documentation**: More accurate examples and validation rules
4. **Workflow Ready**: Structured for LLM function call generation

The API now provides all necessary functionality for building systems thinking models through a clean, well-documented interface suitable for automated prompt workflows.