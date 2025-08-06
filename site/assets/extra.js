/* Custom JavaScript for Convergen documentation */

// Wait for DOM to be ready
document.addEventListener('DOMContentLoaded', function() {
    // Initialize custom features
    initCodeCopyEnhancements();
    initPerformanceMetrics();
    initAccessibilityFeatures();
    initAnalytics();
});

/**
 * Enhance code copy functionality
 */
function initCodeCopyEnhancements() {
    // Add copy success feedback
    document.addEventListener('clipboard-success', function(e) {
        const button = e.detail.trigger;
        const originalText = button.textContent;
        
        button.textContent = 'Copied!';
        button.style.color = '#4caf50';
        
        setTimeout(() => {
            button.textContent = originalText;
            button.style.color = '';
        }, 2000);
    });
    
    // Add annotation highlighting on hover
    const codeBlocks = document.querySelectorAll('code');
    codeBlocks.forEach(block => {
        if (block.textContent.includes(':')) {
            block.addEventListener('mouseenter', function() {
                // Highlight annotation syntax
                const text = this.textContent;
                if (text.startsWith(':')) {
                    this.style.backgroundColor = 'rgba(25, 118, 210, 0.1)';
                    this.style.borderColor = '#1976d2';
                }
            });
            
            block.addEventListener('mouseleave', function() {
                this.style.backgroundColor = '';
                this.style.borderColor = '';
            });
        }
    });
}

/**
 * Display performance metrics and badges
 */
function initPerformanceMetrics() {
    // Add performance badges to relevant sections
    const performanceKeywords = ['40-70% faster', 'concurrent', 'performance'];
    
    document.querySelectorAll('p, li').forEach(element => {
        const text = element.textContent.toLowerCase();
        
        performanceKeywords.forEach(keyword => {
            if (text.includes(keyword)) {
                const regex = new RegExp(`(${keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
                element.innerHTML = element.innerHTML.replace(regex, 
                    '<span class="performance-badge">$1</span>');
            }
        });
    });
}

/**
 * Enhance accessibility features
 */
function initAccessibilityFeatures() {
    // Add skip links for better keyboard navigation
    const skipLink = document.createElement('a');
    skipLink.href = '#main-content';
    skipLink.textContent = 'Skip to main content';
    skipLink.className = 'skip-link';
    skipLink.style.cssText = `
        position: absolute;
        top: -40px;
        left: 6px;
        background: #1976d2;
        color: white;
        padding: 8px;
        text-decoration: none;
        border-radius: 4px;
        z-index: 1000;
        transition: top 0.3s;
    `;
    
    skipLink.addEventListener('focus', function() {
        this.style.top = '6px';
    });
    
    skipLink.addEventListener('blur', function() {
        this.style.top = '-40px';
    });
    
    document.body.insertBefore(skipLink, document.body.firstChild);
    
    // Mark main content area
    const mainContent = document.querySelector('.md-content');
    if (mainContent) {
        mainContent.id = 'main-content';
        mainContent.setAttribute('tabindex', '-1');
    }
    
    // Improve focus indicators for code blocks
    document.querySelectorAll('pre').forEach(pre => {
        pre.setAttribute('tabindex', '0');
        pre.addEventListener('focus', function() {
            this.style.outline = '2px solid #1976d2';
            this.style.outlineOffset = '2px';
        });
        
        pre.addEventListener('blur', function() {
            this.style.outline = '';
            this.style.outlineOffset = '';
        });
    });
}

/**
 * Initialize analytics and user interaction tracking
 */
function initAnalytics() {
    // Track annotation usage in examples
    document.querySelectorAll('code').forEach(code => {
        if (code.textContent.startsWith(':')) {
            code.addEventListener('click', function() {
                // Track which annotations users are most interested in
                const annotation = this.textContent.split(' ')[0];
                
                // Only track if gtag is available (Google Analytics)
                if (typeof gtag !== 'undefined') {
                    gtag('event', 'annotation_click', {
                        'annotation_type': annotation,
                        'page_title': document.title
                    });
                }
            });
        }
    });
    
    // Track example code interactions
    document.querySelectorAll('.highlight').forEach(highlight => {
        highlight.addEventListener('click', function() {
            if (typeof gtag !== 'undefined') {
                gtag('event', 'code_example_view', {
                    'page_title': document.title,
                    'section': this.closest('h1, h2, h3, h4, h5, h6')?.textContent || 'unknown'
                });
            }
        });
    });
}

/**
 * Utility function to show tooltips for annotations
 */
function showAnnotationTooltip(element, annotation) {
    const tooltipTexts = {
        ':match': 'Controls field matching strategy',
        ':skip': 'Excludes fields from conversion',
        ':map': 'Maps fields explicitly',
        ':conv': 'Applies custom converter function',
        ':typecast': 'Enables automatic type casting',
        ':stringer': 'Uses String() methods for conversion',
        ':recv': 'Generates receiver method',
        ':style': 'Controls function signature style',
        ':literal': 'Assigns literal values'
    };
    
    const tooltipText = tooltipTexts[annotation] || 'Convergen annotation';
    
    // Create tooltip element
    const tooltip = document.createElement('div');
    tooltip.textContent = tooltipText;
    tooltip.style.cssText = `
        position: absolute;
        background: #333;
        color: white;
        padding: 6px 10px;
        border-radius: 4px;
        font-size: 12px;
        z-index: 1000;
        pointer-events: none;
        opacity: 0;
        transition: opacity 0.3s;
    `;
    
    document.body.appendChild(tooltip);
    
    // Position tooltip
    const rect = element.getBoundingClientRect();
    tooltip.style.left = rect.left + 'px';
    tooltip.style.top = (rect.top - tooltip.offsetHeight - 5) + 'px';
    tooltip.style.opacity = '1';
    
    // Remove tooltip after delay
    setTimeout(() => {
        tooltip.style.opacity = '0';
        setTimeout(() => {
            document.body.removeChild(tooltip);
        }, 300);
    }, 2000);
}

/**
 * Handle search enhancements
 */
function initSearchEnhancements() {
    const searchInput = document.querySelector('.md-search__input');
    if (searchInput) {
        // Add search suggestions for annotations
        const annotationSuggestions = [
            ':match', ':skip', ':map', ':conv', ':typecast', ':stringer',
            ':recv', ':style', ':literal', ':preprocess', ':postprocess'
        ];
        
        searchInput.addEventListener('input', function() {
            const query = this.value.toLowerCase();
            
            // Show annotation suggestions when typing ':'
            if (query.startsWith(':')) {
                const suggestions = annotationSuggestions.filter(ann => 
                    ann.startsWith(query)
                );
                
                // Display suggestions (implementation would depend on search plugin)
                console.log('Annotation suggestions:', suggestions);
            }
        });
    }
}

// Initialize search enhancements
document.addEventListener('DOMContentLoaded', initSearchEnhancements);

/**
 * Performance monitoring
 */
if ('performance' in window) {
    window.addEventListener('load', function() {
        // Log page load performance
        const perfData = performance.getEntriesByType('navigation')[0];
        
        if (typeof gtag !== 'undefined') {
            gtag('event', 'page_load_time', {
                'load_time': Math.round(perfData.loadEventEnd - perfData.loadEventStart),
                'dom_content_loaded': Math.round(perfData.domContentLoadedEventEnd - perfData.domContentLoadedEventStart)
            });
        }
    });
}