// Code References Page Enhanced Functionality

class CodeReferencesManager {
    constructor() {
        this.currentSection = null;
        this.observer = null;
        this.init();
    }

    init() {
        this.setupSmoothScrolling();
        this.setupActiveNavigation();
        this.setupIntersectionObserver();
        this.setupCodeBlockEnhancements();
        this.setupFileReferences();
        this.addScrollToTopButton();
    }

    setupSmoothScrolling() {
        // Add smooth scrolling to all navigation links
        document.querySelectorAll('.nav-link-custom[href^="#"]').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const targetId = link.getAttribute('href').substring(1);
                const targetElement = document.getElementById(targetId);

                if (targetElement) {
                    const headerOffset = 140; // Account for sticky navbar and padding
                    const elementPosition = targetElement.getBoundingClientRect().top;
                    const offsetPosition = elementPosition + window.pageYOffset - headerOffset;

                    window.scrollTo({
                        top: offsetPosition,
                        behavior: 'smooth'
                    });

                    // Update active state
                    this.setActiveNavItem(link);
                }
            });
        });
    }

    setupActiveNavigation() {
        // Set initial active state based on URL hash
        const hash = window.location.hash;
        if (hash) {
            const activeLink = document.querySelector(`.nav-link-custom[href="${hash}"]`);
            if (activeLink) {
                this.setActiveNavItem(activeLink);
            }
        }

        // Update URL hash when scrolling
        window.addEventListener('hashchange', () => {
            const hash = window.location.hash;
            if (hash) {
                const activeLink = document.querySelector(`.nav-link-custom[href="${hash}"]`);
                if (activeLink) {
                    this.setActiveNavItem(activeLink);
                }
            }
        });
    }

    setupIntersectionObserver() {
        // Create intersection observer to highlight current section
        const observerOptions = {
            root: null,
            rootMargin: '-20% 0px -60% 0px',
            threshold: 0
        };

        this.observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const sectionId = entry.target.id;
                    const navLink = document.querySelector(`.nav-link-custom[href="#${sectionId}"]`);
                    if (navLink) {
                        this.setActiveNavItem(navLink);
                        // Update URL hash without scrolling
                        history.replaceState(null, null, `#${sectionId}`);
                    }
                }
            });
        }, observerOptions);

        // Observe all sections with IDs
        document.querySelectorAll('[id]').forEach(section => {
            if (section.id && section.querySelector('.card')) {
                this.observer.observe(section);
            }
        });
    }

    setActiveNavItem(activeLink) {
        // Remove active class from all nav links
        document.querySelectorAll('.nav-link-custom').forEach(link => {
            link.classList.remove('active');
        });

        // Add active class to current link
        if (activeLink) {
            activeLink.classList.add('active');
            this.currentSection = activeLink.getAttribute('href').substring(1);
        }
    }

    setupCodeBlockEnhancements() {
        // Add copy functionality to code blocks
        document.querySelectorAll('.code-block').forEach((codeBlock, index) => {
            this.enhanceCodeBlock(codeBlock, index);
        });
    }

    enhanceCodeBlock(codeBlock, index) {
        // Create header with copy button
        const header = document.createElement('div');
        header.className = 'code-header';
        header.innerHTML = `
            <div class="code-info">
                <span class="code-language">Go</span>
                <span class="code-lines">${this.countLines(codeBlock)} lines</span>
            </div>
            <button class="copy-code-btn" data-index="${index}">
                <span class="copy-icon">📋</span>
                <span class="copy-text">Copy</span>
            </button>
        `;

        // Insert header before code block
        codeBlock.parentNode.insertBefore(header, codeBlock);

        // Add copy functionality
        const copyBtn = header.querySelector('.copy-code-btn');
        copyBtn.addEventListener('click', () => this.copyCodeBlock(codeBlock, copyBtn));

        // Add line numbers
        this.addLineNumbers(codeBlock);
    }

    countLines(codeBlock) {
        const text = codeBlock.textContent;
        return text.split('\n').length;
    }

    addLineNumbers(codeBlock) {
        const pre = codeBlock.querySelector('pre');
        const code = pre.querySelector('code');

        if (!code) return;

        const lines = code.innerHTML.split('\n');
        const numberedLines = lines.map((line, index) => {
            const lineNumber = index + 1;
            return `<span class="line-number" data-line="${lineNumber}">${lineNumber}</span><span class="line-content">${line}</span>`;
        }).join('\n');

        code.innerHTML = `<div class="line-numbers-wrapper">${numberedLines}</div>`;
        codeBlock.classList.add('has-line-numbers');
    }

    async copyCodeBlock(codeBlock, button) {
        const code = codeBlock.querySelector('code');
        const text = code.textContent;

        try {
            await navigator.clipboard.writeText(text);

            // Update button state
            const icon = button.querySelector('.copy-icon');
            const text_span = button.querySelector('.copy-text');

            icon.textContent = '✅';
            text_span.textContent = 'Copied!';
            button.classList.add('copied');

            // Reset after 2 seconds
            setTimeout(() => {
                icon.textContent = '📋';
                text_span.textContent = 'Copy';
                button.classList.remove('copied');
            }, 2000);

        } catch (err) {
            console.error('Failed to copy code:', err);

            // Fallback for older browsers
            this.fallbackCopyTextToClipboard(text, button);
        }
    }

    fallbackCopyTextToClipboard(text, button) {
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();

        try {
            document.execCommand('copy');
            const icon = button.querySelector('.copy-icon');
            const text_span = button.querySelector('.copy-text');

            icon.textContent = '✅';
            text_span.textContent = 'Copied!';

            setTimeout(() => {
                icon.textContent = '📋';
                text_span.textContent = 'Copy';
            }, 2000);
        } catch (err) {
            console.error('Fallback copy failed:', err);
        }

        document.body.removeChild(textArea);
    }

    setupFileReferences() {
        // Make file path references clickable
        document.querySelectorAll('h5').forEach(heading => {
            const text = heading.textContent;
            if (text.includes('File:')) {
                heading.classList.add('file-reference');
                heading.addEventListener('click', () => {
                    // Extract file path from the heading
                    const codeElement = heading.querySelector('code');
                    if (codeElement) {
                        this.showFileInfo(codeElement.textContent);
                    }
                });
            }
        });
    }

    showFileInfo(filePath) {
        // Create a tooltip or modal with file information
        const tooltip = document.createElement('div');
        tooltip.className = 'file-tooltip';
        tooltip.innerHTML = `
            <div class="file-tooltip-content">
                <h6>📁 ${filePath}</h6>
                <p>Click to view in Cosmos SDK repository</p>
                <div class="file-actions">
                    <button onclick="window.open('https://github.com/cosmos/cosmos-sdk/blob/main/${filePath}', '_blank')" class="btn btn-sm btn-primary">
                        View on GitHub
                    </button>
                </div>
            </div>
        `;

        document.body.appendChild(tooltip);

        // Position tooltip
        const rect = event.target.getBoundingClientRect();
        tooltip.style.cssText = `
            position: absolute;
            top: ${rect.bottom + window.scrollY + 10}px;
            left: ${rect.left + window.scrollX}px;
            z-index: 1000;
        `;

        // Remove tooltip after 5 seconds or on click outside
        setTimeout(() => tooltip.remove(), 5000);

        document.addEventListener('click', function handler(e) {
            if (!tooltip.contains(e.target)) {
                tooltip.remove();
                document.removeEventListener('click', handler);
            }
        });
    }

    addScrollToTopButton() {
        // Create scroll to top button
        const scrollBtn = document.createElement('button');
        scrollBtn.className = 'scroll-to-top';
        scrollBtn.innerHTML = '↑';
        scrollBtn.setAttribute('aria-label', 'Scroll to top');

        document.body.appendChild(scrollBtn);

        // Show/hide button based on scroll position
        window.addEventListener('scroll', () => {
            if (window.pageYOffset > 300) {
                scrollBtn.classList.add('visible');
            } else {
                scrollBtn.classList.remove('visible');
            }
        });

        // Scroll to top on click
        scrollBtn.addEventListener('click', () => {
            window.scrollTo({
                top: 0,
                behavior: 'smooth'
            });
        });
    }
}

// Enhanced Search Functionality
class CodeSearchManager {
    constructor() {
        this.searchResults = [];
        this.currentSearchTerm = '';
        this.init();
    }

    init() {
        this.createSearchBox();
        this.setupSearchFunctionality();
    }

    createSearchBox() {
        const sidebar = document.querySelector('.code-sidebar');
        if (!sidebar) return;

        const searchBox = document.createElement('div');
        searchBox.className = 'search-box';
        searchBox.innerHTML = `
            <div class="search-input-wrapper">
                <input type="text" class="search-input" placeholder="Search code references..." />
                <button class="search-clear" style="display: none;">✕</button>
            </div>
            <div class="search-results" style="display: none;"></div>
        `;

        // Insert after sidebar header
        const sidebarHeader = sidebar.querySelector('.sidebar-header');
        sidebarHeader.insertAdjacentElement('afterend', searchBox);
    }

    setupSearchFunctionality() {
        const searchInput = document.querySelector('.search-input');
        const searchClear = document.querySelector('.search-clear');
        const searchResults = document.querySelector('.search-results');

        if (!searchInput) return;

        // Search as user types
        searchInput.addEventListener('input', (e) => {
            const term = e.target.value.trim();
            this.currentSearchTerm = term;

            if (term.length > 2) {
                this.performSearch(term);
                searchClear.style.display = 'block';
                searchResults.style.display = 'block';
            } else {
                this.clearSearch();
            }
        });

        // Clear search
        searchClear.addEventListener('click', () => {
            searchInput.value = '';
            this.clearSearch();
        });

        // Clear on escape
        searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.clearSearch();
            }
        });
    }

    performSearch(term) {
        const searchResults = document.querySelector('.search-results');
        const allContent = document.querySelectorAll('.card-body, .code-block, h3, h5');

        this.searchResults = [];

        allContent.forEach(element => {
            const text = element.textContent.toLowerCase();
            if (text.includes(term.toLowerCase())) {
                const section = element.closest('[id]');
                if (section) {
                    this.searchResults.push({
                        section: section.id,
                        element: element,
                        preview: this.getPreview(text, term)
                    });
                }
            }
        });

        this.displaySearchResults();
    }

    getPreview(text, term) {
        const index = text.toLowerCase().indexOf(term.toLowerCase());
        const start = Math.max(0, index - 50);
        const end = Math.min(text.length, index + term.length + 50);

        let preview = text.substring(start, end);
        if (start > 0) preview = '...' + preview;
        if (end < text.length) preview = preview + '...';

        // Highlight search term
        const regex = new RegExp(`(${term})`, 'gi');
        return preview.replace(regex, '<mark>$1</mark>');
    }

    displaySearchResults() {
        const searchResults = document.querySelector('.search-results');

        if (this.searchResults.length === 0) {
            searchResults.innerHTML = '<div class="no-results">No results found</div>';
            return;
        }

        const resultsHTML = this.searchResults.map(result => `
            <div class="search-result-item" data-section="${result.section}">
                <div class="result-title">${this.getSectionTitle(result.section)}</div>
                <div class="result-preview">${result.preview}</div>
            </div>
        `).join('');

        searchResults.innerHTML = resultsHTML;

        // Add click handlers
        searchResults.querySelectorAll('.search-result-item').forEach(item => {
            item.addEventListener('click', () => {
                const sectionId = item.dataset.section;
                const element = document.getElementById(sectionId);
                if (element) {
                    const headerOffset = 140;
                    const elementPosition = element.getBoundingClientRect().top;
                    const offsetPosition = elementPosition + window.pageYOffset - headerOffset;

                    window.scrollTo({
                        top: offsetPosition,
                        behavior: 'smooth'
                    });
                }
                this.clearSearch();
            });
        });
    }

    getSectionTitle(sectionId) {
        const section = document.getElementById(sectionId);
        const title = section?.querySelector('h3')?.textContent || sectionId;
        return title;
    }

    clearSearch() {
        const searchInput = document.querySelector('.search-input');
        const searchClear = document.querySelector('.search-clear');
        const searchResults = document.querySelector('.search-results');

        searchClear.style.display = 'none';
        searchResults.style.display = 'none';
        this.searchResults = [];
        this.currentSearchTerm = '';
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new CodeReferencesManager();
    new CodeSearchManager();

    // Add additional CSS for enhanced features
    const enhancedStyles = document.createElement('style');
    enhancedStyles.textContent = `
        .code-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            background: #1e293b;
            padding: 0.75rem 1.5rem;
            border-radius: 1rem 1rem 0 0;
            border-bottom: 1px solid #334155;
            margin-bottom: 0;
        }

        .code-info {
            display: flex;
            gap: 1rem;
            font-size: 0.75rem;
            color: #cbd5e1;
        }

        .code-language {
            background: #3042FB;
            color: white;
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            font-weight: 600;
        }

        .code-lines {
            color: #94a3b8;
        }

        .copy-code-btn {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            background: transparent;
            border: 1px solid #475569;
            color: #cbd5e1;
            padding: 0.375rem 0.75rem;
            border-radius: 0.5rem;
            font-size: 0.75rem;
            cursor: pointer;
            transition: all 0.15s ease;
        }

        .copy-code-btn:hover {
            background: #334155;
            border-color: #3042FB;
            color: white;
        }

        .copy-code-btn.copied {
            background: #059669;
            border-color: #059669;
            color: white;
        }

        .has-line-numbers .line-numbers-wrapper {
            display: table;
            width: 100%;
        }

        .line-number {
            display: table-cell;
            padding-right: 1rem;
            color: #64748b;
            text-align: right;
            user-select: none;
            width: 1%;
            min-width: 2rem;
        }

        .line-content {
            display: table-cell;
            width: 99%;
        }

        .file-tooltip {
            background: var(--surface-color);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            padding: 1rem;
            box-shadow: var(--shadow-xl);
            max-width: 300px;
            animation: fadeIn 0.2s ease-out;
        }

        .scroll-to-top {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            width: 3rem;
            height: 3rem;
            background: var(--gradient-primary);
            color: white;
            border: none;
            border-radius: 50%;
            font-size: 1.25rem;
            font-weight: bold;
            cursor: pointer;
            opacity: 0;
            visibility: hidden;
            transition: all 0.3s ease;
            z-index: 1000;
            box-shadow: var(--shadow-lg);
        }

        .scroll-to-top.visible {
            opacity: 1;
            visibility: visible;
        }

        .scroll-to-top:hover {
            transform: translateY(-2px);
            box-shadow: var(--shadow-xl);
        }

        .search-box {
            margin-bottom: 1.5rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--border-light);
        }

        .search-input-wrapper {
            position: relative;
        }

        .search-input {
            width: 100%;
            padding: 0.75rem 1rem;
            padding-right: 2.5rem;
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            background: var(--surface-color);
            font-size: 0.875rem;
            transition: all 0.15s ease;
        }

        .search-input:focus {
            outline: none;
            border-color: var(--primary-color);
            box-shadow: 0 0 0 0.2rem var(--primary-light);
        }

        .search-clear {
            position: absolute;
            right: 0.75rem;
            top: 50%;
            transform: translateY(-50%);
            background: none;
            border: none;
            color: var(--text-muted);
            cursor: pointer;
            padding: 0.25rem;
            border-radius: 0.25rem;
        }

        .search-clear:hover {
            background: var(--surface-elevated);
        }

        .search-results {
            margin-top: 0.5rem;
            max-height: 300px;
            overflow-y: auto;
            border: 1px solid var(--border-light);
            border-radius: 0.5rem;
            background: var(--surface-color);
        }

        .search-result-item {
            padding: 0.75rem;
            border-bottom: 1px solid var(--border-light);
            cursor: pointer;
            transition: background-color 0.15s ease;
        }

        .search-result-item:hover {
            background: var(--surface-elevated);
        }

        .search-result-item:last-child {
            border-bottom: none;
        }

        .result-title {
            font-weight: 600;
            color: var(--text-primary);
            font-size: 0.875rem;
            margin-bottom: 0.25rem;
        }

        .result-preview {
            font-size: 0.75rem;
            color: var(--text-secondary);
            line-height: 1.4;
        }

        .result-preview mark {
            background: var(--primary-light);
            color: var(--primary-color);
            padding: 0.125rem 0.25rem;
            border-radius: 0.25rem;
        }

        .no-results {
            padding: 1rem;
            text-align: center;
            color: var(--text-muted);
            font-size: 0.875rem;
        }
    `;
    document.head.appendChild(enhancedStyles);

    console.log('🚀 Code References enhanced features loaded!');
});