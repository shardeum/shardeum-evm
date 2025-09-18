// Main JavaScript for Enhanced UI Features

// Dark Mode Toggle Functionality
class ThemeManager {
    constructor() {
        this.currentTheme = localStorage.getItem('theme') || 'light';
        this.init();
    }

    init() {
        this.setTheme(this.currentTheme);
        this.createToggleButton();
        this.bindEvents();
    }

    createToggleButton() {
        const navbar = document.querySelector('.navbar .container');
        if (navbar) {
            const toggleButton = document.createElement('button');
            toggleButton.className = 'theme-toggle ms-3';
            toggleButton.innerHTML = this.currentTheme === 'dark' ? '☀️' : '🌙';
            toggleButton.setAttribute('aria-label', 'Toggle theme');

            // Add to navbar
            const navbarCollapse = navbar.querySelector('.navbar-collapse');
            if (navbarCollapse) {
                navbarCollapse.appendChild(toggleButton);
            } else {
                navbar.appendChild(toggleButton);
            }

            toggleButton.addEventListener('click', () => this.toggleTheme());
        }
    }

    setTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
        this.currentTheme = theme;

        const toggleButton = document.querySelector('.theme-toggle');
        if (toggleButton) {
            toggleButton.innerHTML = theme === 'dark' ? '☀️' : '🌙';
        }
    }

    toggleTheme() {
        const newTheme = this.currentTheme === 'light' ? 'dark' : 'light';
        this.setTheme(newTheme);

        // Add animation effect
        document.body.style.transition = 'all 0.3s ease';
        setTimeout(() => {
            document.body.style.transition = '';
        }, 300);
    }

    bindEvents() {
        // Listen for system theme changes
        if (window.matchMedia) {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
            mediaQuery.addEventListener('change', (e) => {
                if (!localStorage.getItem('theme')) {
                    this.setTheme(e.matches ? 'dark' : 'light');
                }
            });
        }
    }
}

// Enhanced Page Transitions and Animations
class AnimationManager {
    constructor() {
        this.init();
    }

    init() {
        this.animateOnScroll();
        this.enhanceInteractions();
        this.addLoadingStates();
    }

    animateOnScroll() {
        const observerOptions = {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        };

        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('fade-in');
                }
            });
        }, observerOptions);

        // Observe all cards and sections
        document.querySelectorAll('.card, .calculation-section, .legend-panel').forEach(el => {
            observer.observe(el);
        });
    }

    enhanceInteractions() {
        // Add ripple effect to buttons
        document.addEventListener('click', (e) => {
            const button = e.target.closest('.btn');
            if (button) {
                this.createRipple(e, button);
            }
        });

        // Add hover effects to cards
        document.querySelectorAll('.card').forEach(card => {
            card.addEventListener('mouseenter', () => {
                card.style.transform = 'translateY(-4px)';
            });

            card.addEventListener('mouseleave', () => {
                card.style.transform = 'translateY(0)';
            });
        });
    }

    createRipple(event, button) {
        const ripple = document.createElement('span');
        const rect = button.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        const x = event.clientX - rect.left - size / 2;
        const y = event.clientY - rect.top - size / 2;

        ripple.style.cssText = `
            position: absolute;
            width: ${size}px;
            height: ${size}px;
            left: ${x}px;
            top: ${y}px;
            background: rgba(255, 255, 255, 0.3);
            border-radius: 50%;
            transform: scale(0);
            animation: ripple 0.6s linear;
            pointer-events: none;
        `;

        const style = document.createElement('style');
        style.textContent = `
            @keyframes ripple {
                to {
                    transform: scale(2);
                    opacity: 0;
                }
            }
        `;
        document.head.appendChild(style);

        button.style.position = 'relative';
        button.style.overflow = 'hidden';
        button.appendChild(ripple);

        setTimeout(() => {
            ripple.remove();
        }, 600);
    }

    addLoadingStates() {
        // Add loading animation to forms
        document.addEventListener('submit', (e) => {
            const form = e.target;
            const submitBtn = form.querySelector('button[type="submit"]');
            if (submitBtn) {
                submitBtn.classList.add('loading');
                submitBtn.innerHTML = '⏳ Loading...';
            }
        });
    }
}

// Enhanced Form Interactions
class FormEnhancer {
    constructor() {
        this.init();
    }

    init() {
        this.addFloatingLabels();
        this.addValidationFeedback();
        this.enhanceInputs();
    }

    addFloatingLabels() {
        document.querySelectorAll('.form-control').forEach(input => {
            const container = input.parentElement;

            input.addEventListener('focus', () => {
                container.classList.add('focused');
            });

            input.addEventListener('blur', () => {
                if (!input.value) {
                    container.classList.remove('focused');
                }
            });

            // Check if already has value
            if (input.value) {
                container.classList.add('focused');
            }
        });
    }

    addValidationFeedback() {
        document.querySelectorAll('input[type="number"]').forEach(input => {
            input.addEventListener('input', () => {
                this.validateNumberInput(input);
            });
        });
    }

    validateNumberInput(input) {
        const value = parseFloat(input.value);
        const min = parseFloat(input.min);
        const max = parseFloat(input.max);

        input.classList.remove('is-invalid', 'is-valid');

        if (isNaN(value) || (min && value < min) || (max && value > max)) {
            input.classList.add('is-invalid');
        } else {
            input.classList.add('is-valid');
        }
    }

    enhanceInputs() {
        // Add smooth focus transitions
        document.querySelectorAll('.form-control').forEach(input => {
            input.addEventListener('focus', () => {
                input.style.transform = 'scale(1.02)';
            });

            input.addEventListener('blur', () => {
                input.style.transform = 'scale(1)';
            });
        });
    }
}

// Performance Monitoring
class PerformanceMonitor {
    constructor() {
        this.metrics = {};
        this.init();
    }

    init() {
        this.measurePageLoad();
        this.monitorCalculations();
    }

    measurePageLoad() {
        window.addEventListener('load', () => {
            const navigation = performance.getEntriesByType('navigation')[0];
            this.metrics.pageLoad = navigation.loadEventEnd - navigation.loadEventStart;
            console.log(`Page loaded in ${this.metrics.pageLoad}ms`);
        });
    }

    monitorCalculations() {
        if (typeof calculateRewards === 'function') {
            const originalCalculate = calculateRewards;
            window.calculateRewards = () => {
                const start = performance.now();
                originalCalculate();
                const end = performance.now();
                this.metrics.calculationTime = end - start;
                console.log(`Calculation completed in ${this.metrics.calculationTime.toFixed(2)}ms`);
            };
        }
    }
}

// Accessibility Enhancements
class AccessibilityManager {
    constructor() {
        this.init();
    }

    init() {
        this.addKeyboardNavigation();
        this.improveScreenReader();
        this.addSkipLinks();
    }

    addKeyboardNavigation() {
        document.addEventListener('keydown', (e) => {
            // Escape key to close modals/dropdowns
            if (e.key === 'Escape') {
                document.querySelectorAll('.collapse.show').forEach(el => {
                    el.classList.remove('show');
                });
            }

            // Arrow keys for navigation
            if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
                const links = document.querySelectorAll('.nav-link');
                const current = document.activeElement;
                const currentIndex = Array.from(links).indexOf(current);

                if (currentIndex !== -1) {
                    e.preventDefault();
                    const nextIndex = e.key === 'ArrowRight'
                        ? (currentIndex + 1) % links.length
                        : (currentIndex - 1 + links.length) % links.length;
                    links[nextIndex].focus();
                }
            }
        });
    }

    improveScreenReader() {
        // Add ARIA labels to interactive elements
        document.querySelectorAll('.show-calculations-btn').forEach(btn => {
            btn.setAttribute('aria-expanded', 'false');
            btn.addEventListener('click', () => {
                const expanded = btn.getAttribute('aria-expanded') === 'true';
                btn.setAttribute('aria-expanded', !expanded);
            });
        });

        // Add live regions for dynamic content
        const liveRegion = document.createElement('div');
        liveRegion.setAttribute('aria-live', 'polite');
        liveRegion.setAttribute('aria-atomic', 'true');
        liveRegion.className = 'sr-only';
        document.body.appendChild(liveRegion);

        // Announce calculation updates
        if (typeof MutationObserver !== 'undefined') {
            const observer = new MutationObserver(() => {
                liveRegion.textContent = 'Calculations updated';
                setTimeout(() => liveRegion.textContent = '', 1000);
            });

            document.querySelectorAll('.result-value').forEach(el => {
                observer.observe(el, { childList: true, characterData: true });
            });
        }
    }

    addSkipLinks() {
        const skipLink = document.createElement('a');
        skipLink.href = '#main-content';
        skipLink.textContent = 'Skip to main content';
        skipLink.className = 'skip-link sr-only sr-only-focusable';
        skipLink.style.cssText = `
            position: absolute;
            top: -40px;
            left: 6px;
            z-index: 9999;
            background: var(--primary-color);
            color: white;
            padding: 8px 16px;
            text-decoration: none;
            border-radius: 4px;
        `;

        skipLink.addEventListener('focus', () => {
            skipLink.style.top = '6px';
        });

        skipLink.addEventListener('blur', () => {
            skipLink.style.top = '-40px';
        });

        document.body.insertBefore(skipLink, document.body.firstChild);

        // Add ID to main content
        const mainContent = document.querySelector('.main-content');
        if (mainContent) {
            mainContent.id = 'main-content';
        }
    }
}

// Enhanced Error Handling
class ErrorHandler {
    constructor() {
        this.init();
    }

    init() {
        window.addEventListener('error', (e) => {
            this.showError('An unexpected error occurred. Please refresh the page.');
            console.error('Global error:', e.error);
        });

        window.addEventListener('unhandledrejection', (e) => {
            this.showError('A network error occurred. Please check your connection.');
            console.error('Unhandled promise rejection:', e.reason);
        });
    }

    showError(message) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'alert alert-danger fade-in';
        errorDiv.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 9999;
            max-width: 400px;
        `;
        errorDiv.innerHTML = `
            <strong>Error:</strong> ${message}
            <button type="button" class="btn-close" onclick="this.parentElement.remove()"></button>
        `;

        document.body.appendChild(errorDiv);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (errorDiv.parentElement) {
                errorDiv.remove();
            }
        }, 5000);
    }
}

// Progressive Enhancement Features
class ProgressiveEnhancer {
    constructor() {
        this.init();
    }

    init() {
        this.addTooltips();
        this.enhanceCharts();
        this.addCopyToClipboard();
    }

    addTooltips() {
        // Simple tooltip implementation
        document.querySelectorAll('[data-tooltip]').forEach(el => {
            el.addEventListener('mouseenter', (e) => {
                const tooltip = document.createElement('div');
                tooltip.className = 'tooltip-custom';
                tooltip.textContent = e.target.dataset.tooltip;
                tooltip.style.cssText = `
                    position: absolute;
                    background: var(--text-primary);
                    color: var(--surface-color);
                    padding: 8px 12px;
                    border-radius: 6px;
                    font-size: 14px;
                    white-space: nowrap;
                    z-index: 1000;
                    pointer-events: none;
                `;

                document.body.appendChild(tooltip);

                const rect = e.target.getBoundingClientRect();
                tooltip.style.left = rect.left + (rect.width / 2) - (tooltip.offsetWidth / 2) + 'px';
                tooltip.style.top = rect.top - tooltip.offsetHeight - 8 + 'px';
            });

            el.addEventListener('mouseleave', () => {
                document.querySelector('.tooltip-custom')?.remove();
            });
        });
    }

    enhanceCharts() {
        // Add chart interactions if Chart.js is available
        if (typeof Chart !== 'undefined') {
            Chart.defaults.responsive = true;
            Chart.defaults.interaction.intersect = false;
            Chart.defaults.plugins.legend.labels.usePointStyle = true;
        }
    }

    addCopyToClipboard() {
        document.querySelectorAll('.calculation-formula').forEach(formula => {
            const copyBtn = document.createElement('button');
            copyBtn.className = 'btn btn-sm btn-outline-primary copy-btn';
            copyBtn.innerHTML = '📋 Copy';
            copyBtn.style.cssText = `
                position: absolute;
                top: 8px;
                right: 8px;
                font-size: 12px;
                padding: 4px 8px;
            `;

            formula.style.position = 'relative';
            formula.appendChild(copyBtn);

            copyBtn.addEventListener('click', async () => {
                try {
                    await navigator.clipboard.writeText(formula.textContent);
                    copyBtn.innerHTML = '✅ Copied';
                    setTimeout(() => copyBtn.innerHTML = '📋 Copy', 2000);
                } catch (err) {
                    console.error('Copy failed:', err);
                }
            });
        });
    }
}

// Initialize all enhancements when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Initialize all enhancement classes
    new ThemeManager();
    new AnimationManager();
    new FormEnhancer();
    new PerformanceMonitor();
    new AccessibilityManager();
    new ErrorHandler();
    new ProgressiveEnhancer();

    // Add CSS for enhanced styles
    const enhancedStyles = document.createElement('style');
    enhancedStyles.textContent = `
        .focused .form-label {
            transform: translateY(-1.5rem) scale(0.85);
            color: var(--primary-color);
        }

        .form-control.is-valid {
            border-color: var(--success-color);
            box-shadow: 0 0 0 0.2rem rgba(16, 185, 129, 0.25);
        }

        .form-control.is-invalid {
            border-color: var(--danger-color);
            box-shadow: 0 0 0 0.2rem rgba(239, 68, 68, 0.25);
        }

        .sr-only {
            position: absolute !important;
            width: 1px !important;
            height: 1px !important;
            padding: 0 !important;
            margin: -1px !important;
            overflow: hidden !important;
            clip: rect(0, 0, 0, 0) !important;
            white-space: nowrap !important;
            border: 0 !important;
        }

        .sr-only-focusable:focus {
            position: static !important;
            width: auto !important;
            height: auto !important;
            padding: inherit !important;
            margin: inherit !important;
            overflow: visible !important;
            clip: auto !important;
            white-space: inherit !important;
        }

        .copy-btn {
            opacity: 0;
            transition: opacity 0.2s ease;
        }

        .calculation-formula:hover .copy-btn {
            opacity: 1;
        }
    `;
    document.head.appendChild(enhancedStyles);

    console.log('🚀 Enhanced UI features loaded successfully!');
});