function filterTests() {
    const query = document.getElementById('searchInput').value.toLowerCase();
    const testItems = document.querySelectorAll('.test-item');
    const packageSections = document.querySelectorAll('.package-section');
    
    testItems.forEach(item => {
        const text = item.textContent.toLowerCase();
        if (text.includes(query)) {
            item.classList.remove('hidden');
        } else {
            item.classList.add('hidden');
        }
    });
    
    // Show/hide package sections based on whether they have visible tests
    packageSections.forEach(section => {
        const visibleTests = section.querySelectorAll('.test-item:not(.hidden)');
        if (visibleTests.length > 0) {
            section.classList.remove('hidden');
        } else {
            section.classList.add('hidden');
        }
    });
}

function showOnlyFailed() {
    document.getElementById('searchInput').value = '';
    const testItems = document.querySelectorAll('.test-item');
    const packageSections = document.querySelectorAll('.package-section');
    
    testItems.forEach(item => {
        const status = item.dataset.status;
        if (status === 'fail' || status === 'error') {
            item.classList.remove('hidden');
        } else {
            item.classList.add('hidden');
        }
    });
    
    // Show/hide package sections based on whether they have visible failed tests
    packageSections.forEach(section => {
        const visibleTests = section.querySelectorAll('.test-item:not(.hidden)');
        if (visibleTests.length > 0) {
            section.classList.remove('hidden');
        } else {
            section.classList.add('hidden');
        }
    });
    
    // Update button states
    document.getElementById('showAllBtn').classList.remove('active');
    document.getElementById('showFailedBtn').classList.add('active');
}

function showAll() {
    document.getElementById('searchInput').value = '';
    const testItems = document.querySelectorAll('.test-item');
    const packageSections = document.querySelectorAll('.package-section');
    
    testItems.forEach(item => {
        item.classList.remove('hidden');
    });
    
    packageSections.forEach(section => {
        section.classList.remove('hidden');
    });
    
    // Update button states
    document.getElementById('showFailedBtn').classList.remove('active');
    document.getElementById('showAllBtn').classList.add('active');
}

// Organize tests into proper hierarchy after page load
document.addEventListener('DOMContentLoaded', function() {
    organizeTestHierarchy();
});

function organizeTestHierarchy() {
    const hierarchy = document.getElementById('testHierarchy');
    const testItems = Array.from(hierarchy.querySelectorAll('.test-item'));
    
    // If no tests, nothing to organize
    if (testItems.length === 0) {
        return;
    }
    
    // Sort test items by execution order to maintain proper ordering
    testItems.sort((a, b) => {
        const orderA = parseInt(a.dataset.executionOrder) || 0;
        const orderB = parseInt(b.dataset.executionOrder) || 0;
        return orderA - orderB;
    });
    
    // Group tests by package and organize by type
    const packageGroups = {};
    
    testItems.forEach(item => {
        const packageName = item.dataset.package || 'unknown';
        const isSubTest = item.dataset.isSubtest === 'true';
        const parentTest = item.dataset.parentTest || '';
        const testName = item.dataset.name;
        
        if (!packageGroups[packageName]) {
            packageGroups[packageName] = {
                packageTests: [], // Package-level test runners
                individualTests: [], // Individual function tests
                subtests: {}, // Subtests grouped by parent
                orphanSubtests: [] // Subtests without a parent in this package
            };
        }
        
        const group = packageGroups[packageName];
        
        if (isSubTest && parentTest) {
            // This is a subtest - group under its parent
            if (!group.subtests[parentTest]) {
                group.subtests[parentTest] = [];
            }
            group.subtests[parentTest].push(item);
            // Add subtest styling
            item.classList.add('subtest-item');
        } else if (testName && testName.includes('(package)')) {
            // This is a package-level test runner
            group.packageTests.push(item);
        } else {
            // This is an individual test
            group.individualTests.push(item);
        }
    });
    
    // Clear hierarchy and rebuild with proper organization
    hierarchy.innerHTML = '';
    
    // Iterate through packages in a consistent order (alphabetical)
    const sortedPackages = Object.keys(packageGroups).sort();
    
    sortedPackages.forEach(packageName => {
        const group = packageGroups[packageName];
        
        // Skip empty groups
        if (group.packageTests.length === 0 && group.individualTests.length === 0 && Object.keys(group.subtests).length === 0) {
            return;
        }
        
        // Find the package-level test log path for this specific package
        let packageLogPath = '';
        group.packageTests.forEach(packageTest => {
            const logLink = packageTest.querySelector('.log-link');
            if (logLink) {
                packageLogPath = logLink.getAttribute('href');
            }
        });
        
        // Create package section
        const packageSection = document.createElement('div');
        packageSection.className = 'package-section';
        
        // Determine package status based on contained tests
        let hasFailures = false;
        let hasSkipped = false;
        [...group.packageTests, ...group.individualTests, ...Object.values(group.subtests).flat()].forEach(item => {
            if (item.classList.contains('fail') || item.classList.contains('error')) {
                hasFailures = true;
            }
            if (item.classList.contains('skip')) {
                hasSkipped = true;
            }
        });
        
        if (hasFailures) {
            packageSection.classList.add('fail');
        } else if (hasSkipped) {
            packageSection.classList.add('skip');
        }
        
        // Create package header
        const packageHeader = document.createElement('div');
        packageHeader.className = 'package-header';
        if (hasFailures) {
            packageHeader.classList.add('fail');
        } else if (hasSkipped) {
            packageHeader.classList.add('skip');
        }
        
        // Count stats for this package
        let totalTests = group.packageTests.length + group.individualTests.length + Object.values(group.subtests).flat().length;
        let passedTests = 0;
        let failedTests = 0;
        let skippedTests = 0;
        
        [...group.packageTests, ...group.individualTests, ...Object.values(group.subtests).flat()].forEach(item => {
            if (item.classList.contains('pass')) passedTests++;
            else if (item.classList.contains('fail') || item.classList.contains('error')) failedTests++;
            else if (item.classList.contains('skip')) skippedTests++;
        });
        
        packageHeader.innerHTML = `
            <div class="package-name">${packageName}</div>
            <div class="package-stats">
                ${totalTests} tests • 
                ${passedTests} passed • 
                ${failedTests} failed • 
                ${skippedTests} skipped
                ${packageLogPath ? `<a href="${packageLogPath}" target="_blank" class="log-link" style="margin-left: 15px;">View Package Log</a>` : ''}
            </div>
        `;
        
        packageSection.appendChild(packageHeader);
        
        // Add package-level test runners first
        group.packageTests.forEach(test => {
            packageSection.appendChild(test);
        });
        
        // Add individual tests and their subtests
        group.individualTests.forEach(test => {
            packageSection.appendChild(test);
            const testName = test.dataset.name;
            
            // Add any subtests for this test
            if (group.subtests[testName]) {
                group.subtests[testName].sort((a, b) => {
                    const orderA = parseInt(a.dataset.executionOrder) || 0;
                    const orderB = parseInt(b.dataset.executionOrder) || 0;
                    return orderA - orderB;
                });
                group.subtests[testName].forEach(subtest => {
                    packageSection.appendChild(subtest);
                });
                // Mark these subtests as processed
                delete group.subtests[testName];
            }
        });
        
        // Handle orphaned subtests (whose parents aren't in the individual tests list)
        Object.keys(group.subtests).forEach(parentName => {
            const subtests = group.subtests[parentName];
            if (subtests.length === 0) return;
            
            // Sort subtests by execution order
            subtests.sort((a, b) => {
                const orderA = parseInt(a.dataset.executionOrder) || 0;
                const orderB = parseInt(b.dataset.executionOrder) || 0;
                return orderA - orderB;
            });
            
            // Create a virtual parent test header for these orphaned subtests
            const parentHeader = document.createElement('div');
            parentHeader.className = 'test-item virtual-parent';
            
            // Determine the status of the parent based on subtests
            let parentHasFailures = false;
            let parentHasSkipped = false;
            let parentAllPassed = true;
            
            subtests.forEach(subtest => {
                if (subtest.classList.contains('fail') || subtest.classList.contains('error')) {
                    parentHasFailures = true;
                    parentAllPassed = false;
                }
                if (subtest.classList.contains('skip')) {
                    parentHasSkipped = true;
                    parentAllPassed = false;
                }
            });
            
            let parentStatusClass = 'pass';
            let parentStatusText = 'PASS';
            if (parentHasFailures) {
                parentStatusClass = 'fail';
                parentStatusText = 'FAIL';
            } else if (parentHasSkipped) {
                parentStatusClass = 'skip';
                parentStatusText = 'SKIP';
            }
            
            parentHeader.classList.add(parentStatusClass);
            parentHeader.innerHTML = `
                <span class="status-badge ${parentStatusClass}">${parentStatusText}</span>
                <div class="test-name">${parentName}</div>
                <div class="test-details">
                    <span class="package-name">${packageName}</span>
                    <span>Gate: ${subtests[0].dataset.gate || ''}</span>
                    <span class="duration">-</span>
                </div>
            `;
            
            // Add the virtual parent header
            packageSection.appendChild(parentHeader);
            
            // Add all subtests under this parent
            subtests.forEach(subtest => {
                packageSection.appendChild(subtest);
            });
        });
        
        hierarchy.appendChild(packageSection);
    });
} 