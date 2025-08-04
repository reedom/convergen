# Design: MkDocs Documentation Site

## Architecture Overview

This design addresses FR-001 through FR-005 and NFR-001 through NFR-003 from requirements.md.

### Site Structure Design

```
docs/
├── index.md                 # Landing page with overview
├── getting-started/
│   ├── installation.md      # Installation methods
│   ├── quick-start.md       # First steps tutorial
│   └── basic-examples.md    # Simple usage examples
├── guide/
│   ├── annotations.md       # Complete annotation reference
│   ├── advanced-usage.md    # Complex scenarios
│   ├── performance.md       # Performance considerations
│   └── best-practices.md    # Recommended patterns
├── api/
│   ├── cli.md              # Command-line interface
│   ├── go-generate.md      # go:generate integration
│   └── configuration.md    # Configuration options
├── examples/
│   ├── real-world.md       # Production examples
│   ├── generics.md         # Generics examples
│   └── integrations.md     # Framework integrations
└── troubleshooting/
    ├── common-issues.md    # FAQ and solutions
    ├── debugging.md        # Debug techniques
    └── migration.md        # Version migration guide
```

### Technology Stack

**MkDocs Material Features Used:**
- Navigation tabs for major sections
- Search plugin for full-text search
- Code syntax highlighting with Pygments
- Responsive Material Design theme
- Social cards for link previews
- Git revision date plugin for freshness indicators

**GitHub Actions Workflow:**
- Trigger: Push to main branch, manual dispatch
- Steps: Setup Python → Install deps → Build site → Deploy to gh-pages
- Caching: pip dependencies, MkDocs build cache
- Security: GITHUB_TOKEN permissions for pages deployment

## Current State Analysis

**Existing Documentation:**
- README.md: Contains comprehensive annotation table and examples
- TASKS.md: Implementation status and project roadmap  
- tests/: Comprehensive test examples demonstrating features
- .claude/: Detailed project documentation and architecture

**GitHub Integration:**
- Existing workflows: test.yml, code-coverage.yaml
- GitHub Pages: Not currently configured
- Repository settings: Need to enable Pages with Actions source

## Technical Decisions

### MkDocs Configuration Strategy

**Navigation Structure:**
- Use explicit nav configuration for predictable ordering
- Group related content into logical sections
- Maintain shallow hierarchy (max 2 levels) for usability

**Plugin Selection:**
- `search`: Core functionality for content discovery
- `minify`: Asset optimization for performance (NFR-001)
- `git-revision-date-localized`: Show content freshness
- `awesome-pages`: Flexible navigation management

**Theme Customization:**
- Custom CSS for Convergen branding
- Code example styling optimization
- Mobile-first responsive adjustments

### Content Migration Strategy

**README.md Integration:**
- Extract annotation table → guide/annotations.md
- Move examples → examples/ with proper categorization
- Preserve all existing content while improving organization

**Test-Driven Examples:**
- Leverage tests/examples/ for real code examples
- Ensure examples stay current with test suite
- Add explanatory context around test scenarios

### Deployment Architecture

**Build Process:**
1. MkDocs builds static site from docs/ directory
2. GitHub Actions runs on main branch changes
3. Built site deploys to gh-pages branch
4. GitHub Pages serves from gh-pages branch

**Caching Strategy:**
- Cache pip dependencies for faster builds
- Cache MkDocs build artifacts when possible
- Use GitHub Pages CDN for asset delivery

## Integration Points

**With Existing Codebase:**
- Reference actual code files for accuracy
- Link to pkg.go.dev for API documentation
- Maintain consistency with README.md style

**With CI/CD Pipeline:**
- Extend existing GitHub Actions workflows
- Maintain compatibility with current test workflows
- Add documentation build status to PR checks

## Performance Considerations

**Static Site Optimization:**
- Minified CSS/JS assets
- Optimized images with appropriate formats
- Lazy loading for non-critical content

**Search Performance:**
- Client-side search index for instant results
- Search index size optimization
- Progressive search result loading

## Future Extensibility

**Version Management:**
- Design supports future mike plugin integration
- Navigation structure accommodates version selector
- Content organization scales to multiple versions

**Advanced Features:**
- Prepared for API documentation generation
- Structure supports interactive examples
- Design accommodates localization if needed