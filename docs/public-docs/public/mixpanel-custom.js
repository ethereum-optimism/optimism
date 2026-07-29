// Enhanced Mixpanel Analytics for Optimism Docs
// This extends the default Mintlify + Mixpanel integration

(function() {
  // Wait for Mixpanel to be loaded by Mintlify
  function waitForMixpanel(callback) {
    if (typeof mixpanel !== 'undefined') {
      callback();
    } else {
      setTimeout(() => waitForMixpanel(callback), 100);
    }
  }

  waitForMixpanel(() => {
    // Enhanced User Properties
    mixpanel.register({
      'docs_version': 'mintlify_v2',
      'platform': 'optimism_docs',
      'user_agent': navigator.userAgent,
      'screen_resolution': `${screen.width}x${screen.height}`,
      'timezone': Intl.DateTimeFormat().resolvedOptions().timeZone
    });

    // Track Developer Persona Based on Navigation
    function trackDeveloperPersona() {
      const currentTab = document.querySelector('[data-tab].active')?.textContent || 'unknown';
      const personaMap = {
        'App Devs': 'app_developer',
        'Operators': 'chain_operator', 
        'OP Stack': 'protocol_developer',
        'Interoperability': 'interop_developer',
        'Superchain': 'ecosystem_developer'
      };

      const persona = personaMap[currentTab] || 'general_user';
      mixpanel.register({ 'developer_persona': persona });
      
      mixpanel.track('Persona Identified', {
        'persona': persona,
        'tab': currentTab,
        'timestamp': new Date().toISOString()
      });
    }

    // Track Tutorial Progress
    function trackTutorialProgress() {
      const tutorialSteps = document.querySelectorAll('[data-step], .steps > div');
      if (tutorialSteps.length > 0) {
        const currentPage = window.location.pathname;
        
        tutorialSteps.forEach((step, index) => {
          const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
              if (entry.isIntersecting) {
                mixpanel.track('Tutorial Step Viewed', {
                  'tutorial_page': currentPage,
                  'step_number': index + 1,
                  'total_steps': tutorialSteps.length,
                  'progress_percentage': Math.round(((index + 1) / tutorialSteps.length) * 100)
                });
              }
            });
          }, { threshold: 0.5 });
          
          observer.observe(step);
        });
      }
    }

    // Track Code Snippet Interactions
    function trackCodeSnippets() {
      // Track copy button clicks
      document.addEventListener('click', (e) => {
        if (e.target.matches('[data-copy], .copy-button, [aria-label*="Copy"]')) {
          const codeBlock = e.target.closest('pre, .code-block');
          const language = codeBlock?.querySelector('code')?.className?.match(/language-(\w+)/)?.[1] || 'unknown';
          
          mixpanel.track('Code Copied', {
            'language': language,
            'page': window.location.pathname,
            'code_length': codeBlock?.textContent?.length || 0
          });
        }
      });
    }

    // Track AI Feature Usage
    function trackAIFeatures() {
      // Track contextual actions (copy to ChatGPT, Claude, etc.)
      document.addEventListener('click', (e) => {
        const button = e.target.closest('[data-action]');
        if (button) {
          const action = button.getAttribute('data-action') || button.textContent.toLowerCase();
          
          if (action.includes('chatgpt') || action.includes('claude') || action.includes('copy')) {
            mixpanel.track('AI Feature Used', {
              'feature_type': action,
              'page': window.location.pathname,
              'content_type': determineContentType()
            });
          }
        }
      });
    }

    // Track Search Behavior
    function trackSearchBehavior() {
      const searchInput = document.querySelector('[data-search], input[type="search"]');
      if (searchInput) {
        let searchTimeout;
        
        searchInput.addEventListener('input', (e) => {
          clearTimeout(searchTimeout);
          searchTimeout = setTimeout(() => {
            if (e.target.value.length > 2) {
              mixpanel.track('Search Query', {
                'query': e.target.value,
                'query_length': e.target.value.length,
                'page_context': window.location.pathname
              });
            }
          }, 1000);
        });
      }
    }

    // Track Feedback Interactions
    function trackFeedback() {
      document.addEventListener('click', (e) => {
        // Thumbs rating
        if (e.target.matches('[data-rating], .thumbs-up, .thumbs-down')) {
          const rating = e.target.classList.contains('thumbs-up') || 
                        e.target.getAttribute('data-rating') === 'positive' ? 'positive' : 'negative';
          
          mixpanel.track('Page Feedback', {
            'rating': rating,
            'page': window.location.pathname,
            'content_type': determineContentType()
          });
        }

        // Suggest edit / Raise issue
        if (e.target.matches('[data-suggest-edit], [data-raise-issue]')) {
          const actionType = e.target.getAttribute('data-suggest-edit') !== null ? 'suggest_edit' : 'raise_issue';
          
          mixpanel.track('Content Improvement Action', {
            'action_type': actionType,
            'page': window.location.pathname,
            'content_type': determineContentType()
          });
        }
      });
    }

    // Helper function to determine content type
    function determineContentType() {
      const path = window.location.pathname;
      if (path.includes('/tutorials/')) return 'tutorial';
      if (path.includes('/tools/')) return 'tool_documentation';
      if (path.includes('/configuration/')) return 'configuration_guide';
      if (path.includes('/api/')) return 'api_reference';
      return 'general_documentation';
    }

    // Track Time on Page
    let pageStartTime = Date.now();
    let isPageVisible = true;

    document.addEventListener('visibilitychange', () => {
      isPageVisible = !document.hidden;
    });

    window.addEventListener('beforeunload', () => {
      if (isPageVisible) {
        const timeOnPage = Math.round((Date.now() - pageStartTime) / 1000);
        mixpanel.track('Page Session End', {
          'time_on_page_seconds': timeOnPage,
          'page': window.location.pathname,
          'content_type': determineContentType()
        });
      }
    });

    // Initialize all tracking
    function initializeTracking() {
      trackDeveloperPersona();
      trackTutorialProgress();
      trackCodeSnippets();
      trackAIFeatures();
      trackSearchBehavior();
      trackFeedback();
      
      // Track initial page view with enhanced properties
      mixpanel.track('Enhanced Page View', {
        'page': window.location.pathname,
        'content_type': determineContentType(),
        'referrer': document.referrer,
        'timestamp': new Date().toISOString()
      });
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', initializeTracking);
    } else {
      initializeTracking();
    }

    // Re-initialize on navigation (SPA behavior)
    let currentPath = window.location.pathname;
    setInterval(() => {
      if (window.location.pathname !== currentPath) {
        currentPath = window.location.pathname;
        pageStartTime = Date.now();
        setTimeout(initializeTracking, 500); // Small delay for DOM updates
      }
    }, 1000);
  });
})();
