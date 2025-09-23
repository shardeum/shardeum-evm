// Cosmos Rewards Calculator JavaScript - Updated

// Global variables
let inflationChart = null;

// Initialize calculator on page load
document.addEventListener('DOMContentLoaded', function() {
    // Add event listeners to all input fields
    const inputs = document.querySelectorAll('input[type="number"]');
    inputs.forEach(input => {
        input.addEventListener('input', calculateRewards);
    });

    // Initial calculation
    calculateRewards();
    createInflationChart();
});

// Main calculation function
function calculateRewards() {
    // Get all input values
    const params = {
        inflationRateChange: parseFloat(document.getElementById('inflationRateChange').value) / 100,
        inflationMax: parseFloat(document.getElementById('inflationMax').value) / 100,
        inflationMin: parseFloat(document.getElementById('inflationMin').value) / 100,
        goalBonded: parseFloat(document.getElementById('goalBonded').value) / 100,
        initialInflation: parseFloat(document.getElementById('initialInflation').value) / 100,
        blocksPerYear: parseFloat(document.getElementById('blocksPerYear').value),
        currentBonded: parseFloat(document.getElementById('currentBonded').value) / 100,
        totalSupply: parseFloat(document.getElementById('totalSupply').value),
        validatorCommission: parseFloat(document.getElementById('validatorCommission').value) / 100
    };

    // Calculate current inflation rate
    const currentInflation = calculateCurrentInflation(params);

    // Calculate per-block change
    const perBlockChange = calculatePerBlockChange(params);

    // Calculate timeline projections
    const timeline = calculateTimeline(params, currentInflation, perBlockChange);

    // Calculate annual provisions
    const provisions = calculateAnnualProvisions(params, currentInflation);

    // Calculate block provisions
    const blockProvisions = calculateBlockProvisions(provisions, params);

    // Calculate APR
    const apr = calculateAPR(params, provisions);

    // Update UI
    updateCurrentInflationDisplay(params, currentInflation);
    updateInflationDisplay(currentInflation, perBlockChange);
    updateTimelineDisplay(timeline);
    updateAnnualProvisionsDisplay(provisions, params);
    updateBlockProvisionsDisplay(blockProvisions, params);
    updateAPRDisplay(apr);
    updateCalculationDetails(params, currentInflation, perBlockChange, provisions, apr);

    // Update charts
    updateInflationChart(params);
    updateProvisionsChart(params);
}

// Calculate current inflation rate using actual Cosmos SDK NextInflationRate logic
function calculateCurrentInflation(params) {
    // The Cosmos SDK NextInflationRate function calculates the NEXT block's inflation
    // based on the current bonding ratio, not a cumulative adjustment
    
    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases
        // Formula: inflation_rate_change × (goal_bonded - current_bonded) / blocks_per_year
        const gap = params.goalBonded - params.currentBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        const newInflation = params.initialInflation + perBlockAdjustment;
        return Math.min(newInflation, params.inflationMax);
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases
        // Formula: inflation_rate_change × (current_bonded - goal_bonded) / blocks_per_year
        const gap = params.currentBonded - params.goalBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        const newInflation = params.initialInflation - perBlockAdjustment;
        return Math.max(newInflation, params.inflationMin);
    } else {
        // At target: no change from initial
        return params.initialInflation;
    }
}

// Calculate per-block change using correct Cosmos SDK formula
function calculatePerBlockChange(params) {
    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases (positive change)
        // Formula: (inflationRateChange * gap) / blocksPerYear
        const gap = params.goalBonded - params.currentBonded;
        return (params.inflationRateChange * gap) / params.blocksPerYear;
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases (negative change)
        // Formula: -(inflationRateChange * gap) / blocksPerYear
        const gap = params.currentBonded - params.goalBonded;
        return -(params.inflationRateChange * gap) / params.blocksPerYear;
    } else {
        // At target: no change
        return 0;
    }
}

// Calculate timeline projections
function calculateTimeline(params, currentInflation, perBlockChange) {
    // Calculate inflation after specific time periods
    // Note: This assumes bonding ratio stays constant (simplified model)
    const blocksPerDay = 17280; // ~5 second blocks
    const blocksPerWeek = 120960; // ~7 days
    const blocksPerMonth = 525960; // ~30.44 days per month
    
    // Helper function to apply directional bounds
    const applyBounds = (rate) => {
        if (perBlockChange > 0) {
            // Under-staked: inflation increases, only apply max bound
            return Math.min(params.inflationMax, rate);
        } else if (perBlockChange < 0) {
            // Over-staked: inflation decreases, only apply min bound
            return Math.max(params.inflationMin, rate);
        }
        return rate; // No change
    };
    
    return {
        block1: currentInflation,
        day1: applyBounds(currentInflation + (perBlockChange * blocksPerDay)),
        week1: applyBounds(currentInflation + (perBlockChange * blocksPerWeek)),
        month1: applyBounds(currentInflation + (perBlockChange * blocksPerMonth)),
        year1: applyBounds(currentInflation + (perBlockChange * params.blocksPerYear))
    };
}

// Calculate annual provisions
function calculateAnnualProvisions(params, currentInflation) {
    const totalMinted = currentInflation * params.totalSupply;
    const communityTax = 0.02; // 2% community tax
    const communityPool = totalMinted * communityTax;
    const validatorRewards = totalMinted - communityPool;

    return {
        totalMinted,
        communityPool,
        validatorRewards
    };
}

// Calculate block provisions
function calculateBlockProvisions(provisions, params) {
    const perBlockProvision = provisions.validatorRewards / params.blocksPerYear;
    const blockTime = 31536000 / params.blocksPerYear; // seconds per year / blocks per year

    return {
        perBlockProvision,
        blockTime
    };
}

// Calculate APR using correct Cosmos SDK logic
function calculateAPR(params, provisions) {
    const totalStaked = params.totalSupply * params.currentBonded;
    
    // Base APR is the total validator rewards divided by total staked
    const baseAPR = provisions.validatorRewards / totalStaked;
    
    // Delegator APR is base APR minus validator commission
    const delegatorAPR = baseAPR * (1 - params.validatorCommission);
    
    // Validator APR includes:
    // 1. Full base APR on their own stake
    // 2. Commission from delegators' rewards
    // For simplicity, we show the effective APR a validator earns
    const validatorAPR = baseAPR; // This represents the full rate validators earn
    
    // Commission earnings calculation
    // Validators earn commission on delegator rewards, not total rewards
    const delegatorRewards = provisions.validatorRewards * (1 - params.validatorCommission);
    const commissionEarnings = delegatorRewards * params.validatorCommission / (1 - params.validatorCommission);

    return {
        baseAPR,
        validatorAPR,
        delegatorAPR,
        commissionEarnings,
        totalStaked
    };
}

// Update current inflation calculation display
function updateCurrentInflationDisplay(params, currentInflation) {
    const gap = Math.abs(params.currentBonded - params.goalBonded);
    const gapSign = params.currentBonded < params.goalBonded ? '-' : '+';
    const direction = params.currentBonded < params.goalBonded ? 'Increasing' :
                     params.currentBonded > params.goalBonded ? 'Decreasing' : 'Stable';

    // Calculate per-block adjustment using correct Cosmos SDK formula
    let perBlockAdjustment;
    const baseRate = params.initialInflation; // Use actual initial inflation from genesis
    
    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases
        const gap = params.goalBonded - params.currentBonded;
        perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases
        const gap = params.currentBonded - params.goalBonded;
        perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
    } else {
        perBlockAdjustment = 0;
    }
    
    const calculatedRate = params.currentBonded < params.goalBonded ?
                          baseRate + perBlockAdjustment :
                          params.currentBonded > params.goalBonded ?
                          baseRate - perBlockAdjustment : baseRate;

    const boundedRate = Math.max(params.inflationMin, Math.min(params.inflationMax, calculatedRate));

    // Update main displays
    document.getElementById('actualCurrentInflation').textContent = (currentInflation * 100).toFixed(2) + '%';
    document.getElementById('bondingGap').textContent = gapSign + (gap * 100).toFixed(1) + '%';
    document.getElementById('inflationDirection').textContent = direction;

    // Update calculation details with safety checks
    const updateElement = (id, value) => {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value;
        } else {
            console.warn(`Element with id '${id}' not found`);
        }
    };
    
    updateElement('calc-current', (params.currentBonded * 100).toFixed(0));
    updateElement('calc-goal', (params.goalBonded * 100).toFixed(0));
    updateElement('calc-gap', (gap * 100).toFixed(0));
    updateElement('calc-rate-change', (params.inflationRateChange * 100).toFixed(0));
    updateElement('calc-gap-2', (gap * 100).toFixed(0));

    // Calculate per-block adjustment correctly (reuse existing variable)
    // perBlockAdjustment is already calculated above with correct sign
    updateElement('calc-per-block', (perBlockAdjustment * 100).toFixed(8) + '%');

    // Add blocks per year to display
    updateElement('calc-blocks', params.blocksPerYear.toLocaleString());
    updateElement('calc-direction', (perBlockAdjustment * 100).toFixed(8) + '%');

    // Update formula direction using per-block adjustment with dynamic values
    const formulaElement = document.getElementById('adjustment-formula');
    if (formulaElement) {
        if (params.currentBonded < params.goalBonded) {
            formulaElement.innerHTML = `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) &lt; Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>Add</strong> per-block adjustment<br>
                                       Next Block Inflation = ${(baseRate * 100).toFixed(3)}% + ${(perBlockAdjustment * 100).toFixed(8)}% = <span id="final-inflation-calc">${(calculatedRate * 100).toFixed(6)}%</span>`;
        } else if (params.currentBonded > params.goalBonded) {
            formulaElement.innerHTML = `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) &gt; Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>Subtract</strong> per-block adjustment<br>
                                       Next Block Inflation = ${(baseRate * 100).toFixed(3)}% - ${(Math.abs(perBlockAdjustment) * 100).toFixed(8)}% = <span id="final-inflation-calc">${(calculatedRate * 100).toFixed(6)}%</span>`;
        } else {
            formulaElement.innerHTML = `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) = Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>No</strong> adjustment<br>
                                       Next Block Inflation = ${(baseRate * 100).toFixed(3)}% + 0% = <span id="final-inflation-calc">${(baseRate * 100).toFixed(3)}%</span>`;
        }
    }
}

// Update inflation display
function updateInflationDisplay(currentInflation, perBlockChange) {
    document.getElementById('currentInflation').textContent = (currentInflation * 100).toFixed(2) + '%';
    document.getElementById('perBlockChange').textContent =
        (perBlockChange >= 0 ? '+' : '') + (perBlockChange * 100).toFixed(8) + '%';
}

// Update timeline display
function updateTimelineDisplay(timeline) {
    document.getElementById('block-1').textContent = `Inflation: ${(timeline.block1 * 100).toFixed(6)}%`;
    document.getElementById('block-day').textContent = `Inflation: ${(timeline.day1 * 100).toFixed(6)}%`;
    document.getElementById('block-week').textContent = `Inflation: ${(timeline.week1 * 100).toFixed(6)}%`;
    document.getElementById('block-month').textContent = `Inflation: ${(timeline.month1 * 100).toFixed(6)}%`;
    document.getElementById('block-year').textContent = `Inflation: ${(timeline.year1 * 100).toFixed(2)}% (theoretical max)`;
}

// Update annual provisions display with time period support
function updateAnnualProvisionsDisplay(provisions, params) {
    // Calculate time-based provisions
    const timeBasedProvisions = calculateTimeBasedProvisions(provisions, params, currentProvisionPeriod);
    
    // Update labels based on period
    const labels = getProvisionLabels(currentProvisionPeriod);
    
    document.getElementById('totalMinted').textContent = formatLargeNumber(timeBasedProvisions.totalMinted);
    document.getElementById('communityPool').textContent = formatLargeNumber(timeBasedProvisions.communityPool);
    document.getElementById('validatorRewards').textContent = formatLargeNumber(timeBasedProvisions.validatorRewards);
    
    // Update labels
    document.querySelector('#totalMinted').nextElementSibling.textContent = labels.totalMinted;
    document.querySelector('#communityPool').nextElementSibling.textContent = labels.communityPool;
    document.querySelector('#validatorRewards').nextElementSibling.textContent = labels.validatorRewards;
}

// Update block provisions display
function updateBlockProvisionsDisplay(blockProvisions, params) {
    document.getElementById('perBlockProvision').textContent = Math.round(blockProvisions.perBlockProvision).toLocaleString();
    document.getElementById('blockTime').textContent = `~${blockProvisions.blockTime.toFixed(1)} sec`;
}

// Update APR display
function updateAPRDisplay(apr) {
    document.getElementById('validatorAPR').textContent = (apr.validatorAPR * 100).toFixed(1) + '%';
    document.getElementById('delegatorAPR').textContent = (apr.delegatorAPR * 100).toFixed(1) + '%';
    document.getElementById('commissionEarnings').textContent = formatLargeNumber(apr.commissionEarnings);
}

// Update calculation details
function updateCalculationDetails(params, currentInflation, perBlockChange, provisions, apr) {
    console.log('Updating calculation details with params:', params);
    console.log('Current bonded:', params.currentBonded * 100, 'Goal bonded:', params.goalBonded * 100);
    
    // Helper function for safe element updates
    const updateElement = (id, value) => {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value;
        } else {
            console.warn(`Element with id '${id}' not found`);
        }
    };
    
    // Inflation calculation details
    const gap = Math.abs(params.goalBonded - params.currentBonded) * 100;
    const gapRatio = gap / (params.goalBonded * 100);

    // Update the gap calculation text to show correct values
    const gapCalculationText1 = `Gap = |Current Bonded - Goal Bonded| = |${(params.currentBonded * 100).toFixed(0)}% - ${(params.goalBonded * 100).toFixed(0)}%| = ${gap.toFixed(0)}%`;
    const gapCalculationText2 = `Gap = |Goal Bonded - Current Bonded| = |${(params.goalBonded * 100).toFixed(0)}% - ${(params.currentBonded * 100).toFixed(0)}%| = ${gap.toFixed(0)}%`;
    
    // Find and update the gap calculation steps
    const calculationSteps = document.querySelectorAll('.calculation-step');
    console.log('Found calculation steps:', calculationSteps.length);
    
    calculationSteps.forEach((step, index) => {
        if (step.textContent.includes('Calculate bonding gap')) {
            console.log('Found bonding gap step:', index, step.textContent);
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                console.log('Found formula element, updating...');
                // Check which format this section uses
                if (formulaElement.textContent.includes('Current Bonded - Goal Bonded')) {
                    formulaElement.innerHTML = gapCalculationText1;
                } else {
                    formulaElement.innerHTML = gapCalculationText2;
                }
            } else {
                console.log('No formula element found in step:', index);
            }
        }
    });

    // Update all per-block change displays
    const perBlockChangeValue = (perBlockChange * 100).toFixed(8);
    const perBlockChangeDisplay = perBlockChange >= 0 ? `+${perBlockChangeValue}%` : `${perBlockChangeValue}%`;
    
    // Update main per-block change display
    const perBlockElement = document.getElementById('perBlockChange');
    if (perBlockElement) {
        perBlockElement.textContent = perBlockChangeDisplay;
    }
    
    // Update the direction text dynamically
    const directionText = params.currentBonded < params.goalBonded ? 
        'Since Current Bonded < Goal Bonded: <strong>Inflation increases</strong>' :
        params.currentBonded > params.goalBonded ?
        'Since Current Bonded > Goal Bonded: <strong>Inflation decreases</strong>' :
        'Since Current Bonded = Goal Bonded: <strong>No change</strong>';
    
    // Update gap calculation text (correct Cosmos SDK formula)
    const gapText = `Gap = |Goal Bonded - Current Bonded| = |${(params.goalBonded * 100).toFixed(0)}% - ${(params.currentBonded * 100).toFixed(0)}%| = ${gap.toFixed(0)}%`;
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Calculate bonding gap')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                formulaElement.innerHTML = gapText;
            }
        }
    });

    // Update per-block change calculation text (matching Go code exactly)
    const gapValue = Math.abs(params.goalBonded - params.currentBonded);
    const perBlockChangeText = `Per-Block Change = (Inflation Rate Change × Gap) / Blocks Per Year<br>= (${(params.inflationRateChange * 100).toFixed(0)}% × ${(gapValue * 100).toFixed(0)}%) / ${params.blocksPerYear.toLocaleString()}<br>= ${(Math.abs(perBlockChange) * 100).toFixed(8)}% per block`;
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Calculate per-block change')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                formulaElement.innerHTML = perBlockChangeText;
            }
        }
    });

    // Find and update the direction step with dynamic text
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Determine direction')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                const changeSign = perBlockChange >= 0 ? '+' : '';
                const dynamicDirectionText = params.currentBonded < params.goalBonded ? 
                    `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) < Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>Inflation increases</strong>` :
                    params.currentBonded > params.goalBonded ?
                    `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) > Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>Inflation decreases</strong>` :
                    `Since Current Bonded (${(params.currentBonded * 100).toFixed(0)}%) = Goal Bonded (${(params.goalBonded * 100).toFixed(0)}%): <strong>No change</strong>`;
                
                formulaElement.innerHTML = `${dynamicDirectionText} by <span id="calc-direction">${changeSign}${(perBlockChange * 100).toFixed(8)}%</span> per block`;
            }
        }
    });

    // Timeline calculation details - make dynamic
    updateElement('timeline-per-block', (perBlockChange * 100).toFixed(8) + '%');
    
    // Update timeline calculation formula dynamically
    const timelineFormulaElement = document.querySelector('#timeline-calc .calculation-formula');
    if (timelineFormulaElement) {
        const blocksPerDay = 17280;
        const day1Change = perBlockChange * blocksPerDay;
        const day1Inflation = params.initialInflation + day1Change;
        timelineFormulaElement.innerHTML = `Day 1: ${(params.initialInflation * 100).toFixed(2)}% + (<span id="timeline-per-block">${(perBlockChange * 100).toFixed(8)}%</span> × ${blocksPerDay.toLocaleString()}) = ${(params.initialInflation * 100).toFixed(2)}% + ${(day1Change * 100).toFixed(6)}% = ${(day1Inflation * 100).toFixed(6)}%`;
    }

    // Annual provisions calculation details
    updateElement('annual-inflation', (currentInflation * 100).toFixed(1) + '%');
    updateElement('annual-supply', params.totalSupply.toLocaleString());
    updateElement('annual-total', Math.round(provisions.totalMinted).toLocaleString());
    updateElement('annual-total-2', Math.round(provisions.totalMinted).toLocaleString());
    updateElement('annual-community', Math.round(provisions.communityPool).toLocaleString());
    updateElement('annual-total-3', Math.round(provisions.totalMinted).toLocaleString());
    updateElement('annual-community-2', Math.round(provisions.communityPool).toLocaleString());
    updateElement('annual-validators', Math.round(provisions.validatorRewards).toLocaleString());

    // Block provisions calculation details
    updateElement('block-annual', Math.round(provisions.validatorRewards).toLocaleString());
    updateElement('block-blocks', params.blocksPerYear.toLocaleString());
    updateElement('block-result', Math.round(provisions.validatorRewards / params.blocksPerYear).toLocaleString());
    updateElement('block-blocks-2', params.blocksPerYear.toLocaleString());
    updateElement('block-time-calc', (31536000 / params.blocksPerYear).toFixed(2));

    // APR calculation details
    updateElement('apr-supply', params.totalSupply.toLocaleString());
    updateElement('apr-bonded', (params.currentBonded * 100).toFixed(0) + '%');
    updateElement('apr-staked', Math.round(apr.totalStaked).toLocaleString());
    updateElement('apr-rewards', Math.round(provisions.validatorRewards).toLocaleString());
    updateElement('apr-staked-2', Math.round(apr.totalStaked).toLocaleString());
    updateElement('apr-base', (apr.baseAPR * 100).toFixed(2) + '%');
    updateElement('apr-base-2', (apr.baseAPR * 100).toFixed(2) + '%');
    updateElement('apr-commission', (params.validatorCommission * 100).toFixed(0) + '%');
    updateElement('apr-delegator-calc', (apr.delegatorAPR * 100).toFixed(2) + '%');
}

// Toggle calculation details
function toggleCalculations(sectionId) {
    const section = document.getElementById(sectionId);
    if (section.style.display === 'none') {
        section.style.display = 'block';
        section.classList.add('fade-in');
    } else {
        section.style.display = 'none';
    }
}

// Make functions globally available
window.toggleCalculations = toggleCalculations;

// Load preset scenarios
function loadScenario(scenario) {
    switch(scenario) {
        case 'under-staked':
            document.getElementById('currentBonded').value = 40;
            document.getElementById('inflationRateChange').value = 13;
            document.getElementById('inflationMax').value = 20;
            document.getElementById('inflationMin').value = 7;
            document.getElementById('goalBonded').value = 67;
            break;
        case 'target':
            document.getElementById('currentBonded').value = 67;
            document.getElementById('inflationRateChange').value = 13;
            document.getElementById('inflationMax').value = 20;
            document.getElementById('inflationMin').value = 7;
            document.getElementById('goalBonded').value = 67;
            break;
        case 'over-staked':
            document.getElementById('currentBonded').value = 85;
            document.getElementById('inflationRateChange').value = 13;
            document.getElementById('inflationMax').value = 20;
            document.getElementById('inflationMin').value = 7;
            document.getElementById('goalBonded').value = 67;
            break;
    }
    calculateRewards();
}

// Make functions globally available
window.loadScenario = loadScenario;

// Create inflation chart
function createInflationChart() {
    const ctx = document.getElementById('inflationChart');
    if (!ctx) {
        console.warn('Chart canvas not found');
        return;
    }

    try {
        const context = ctx.getContext('2d');

    inflationChart = new Chart(context, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Inflation Rate (%)',
                data: [],
                borderColor: '#3042FB',
                backgroundColor: 'rgba(48, 66, 251, 0.08)',
                borderWidth: 3,
                fill: true,
                tension: 0.4,
                pointBackgroundColor: '#3042FB',
                pointBorderColor: '#ffffff',
                pointBorderWidth: 2,
                pointRadius: 5,
                pointHoverRadius: 8
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                intersect: false,
                mode: 'index'
            },
            plugins: {
                legend: {
                    display: true,
                    position: 'top',
                    labels: {
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#4a5568',
                        usePointStyle: true,
                        pointStyle: 'circle',
                        padding: 20
                    }
                },
                title: {
                    display: true,
                    text: 'Inflation Rate Over Time',
                    font: {
                        family: 'Space Grotesk',
                        size: 18,
                        weight: '700'
                    },
                    color: '#0f1419',
                    padding: {
                        top: 10,
                        bottom: 30
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(15, 20, 25, 0.9)',
                    titleColor: '#ffffff',
                    bodyColor: '#cbd5e1',
                    borderColor: '#3042FB',
                    borderWidth: 1,
                    cornerRadius: 8,
                    displayColors: false,
                    titleFont: {
                        family: 'Inter',
                        size: 14,
                        weight: '600'
                    },
                    bodyFont: {
                        family: 'Inter',
                        size: 13
                    },
                    callbacks: {
                        title: function(context) {
                            return `Block: ${context[0].label}`;
                        },
                        label: function(context) {
                            return `Inflation Rate: ${context.parsed.y.toFixed(2)}%`;
                        }
                    }
                }
            },
            scales: {
                x: {
                    title: {
                        display: true,
                        text: 'Bonding Ratio (%)',
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#4a5568'
                    },
                    grid: {
                        color: 'rgba(203, 213, 225, 0.3)',
                        drawBorder: false
                    },
                    ticks: {
                        font: {
                            family: 'JetBrains Mono',
                            size: 12
                        },
                        color: '#718096'
                    }
                },
                y: {
                    title: {
                        display: true,
                        text: 'Inflation Rate (%)',
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#4a5568'
                    },
                    min: 0,
                    max: 25,
                    grid: {
                        color: 'rgba(203, 213, 225, 0.3)',
                        drawBorder: false
                    },
                    ticks: {
                        font: {
                            family: 'JetBrains Mono',
                            size: 12
                        },
                        color: '#718096',
                        callback: function(value) {
                            return value + '%';
                        }
                    }
                }
            }
        }
    });

        // Initialize chart with default data
        updateInflationChart({
            inflationRateChange: 0.13,
            inflationMax: 0.20,
            inflationMin: 0.07,
            goalBonded: 0.67,
            initialInflation: 0.13,
            currentBonded: 0.50,
            blocksPerYear: 6311520
        });

    } catch (error) {
        console.error('Failed to create inflation chart:', error);
        inflationChart = null;
    }
}

// Global variables for current periods
let currentChartPeriod = 'blocks';
let currentProvisionPeriod = 'annual';

// Update inflation chart with time-based data
function updateInflationChart(params) {
    if (!inflationChart || !inflationChart.data) return;

    // Use initial inflation rate from input instead of calculated current
    const startingInflation = params.initialInflation || 0.13;
    const perBlockChange = calculatePerBlockChangeForBonding(params, params.currentBonded);

    const chartData = generateTimeBasedData(params, startingInflation, perBlockChange, currentChartPeriod);

    inflationChart.data.labels = chartData.labels;
    inflationChart.data.datasets[0].data = chartData.rates;

    // Update x-axis title based on period
    const xAxisTitle = {
        blocks: 'Block Number (×1000)',
        hours: 'Hours',
        days: 'Days',
        months: 'Months'
    };

    inflationChart.options.scales.x.title.text = xAxisTitle[currentChartPeriod];
    inflationChart.update();

    // Update chart stats
    updateChartStats(chartData.rates);
}

// Calculate per-block change for a specific bonding ratio (matching Go code exactly)
function calculatePerBlockChangeForBonding(params, bondingRatio) {
    if (bondingRatio < params.goalBonded) {
        // Under-staked: inflation increases
        // Formula: (inflationRateChange * gap) / blocksPerYear
        const gap = params.goalBonded - bondingRatio;
        return (params.inflationRateChange * gap) / params.blocksPerYear;
    } else if (bondingRatio > params.goalBonded) {
        // Over-staked: inflation decreases
        // Formula: -(inflationRateChange * gap) / blocksPerYear
        const gap = bondingRatio - params.goalBonded;
        return -(params.inflationRateChange * gap) / params.blocksPerYear;
    } else {
        // At target: no change
        return 0;
    }
}

// Generate time-based data for different periods
function generateTimeBasedData(params, startInflation, perBlockChange, period) {
    const labels = [];
    const rates = [];

    let inflation = startInflation;

    switch (period) {
        case 'blocks':
            // Show first 100,000 blocks to see meaningful change
            for (let block = 0; block <= 100000; block += 1000) {
                labels.push(block);
                rates.push(inflation * 100);

                // Calculate inflation for next 1000 blocks
                for (let i = 0; i < 1000; i++) {
                    inflation += perBlockChange;
                }
                // Apply bounds check after the full period, not per block
                // Only apply the relevant bound based on direction
                if (perBlockChange > 0) {
                    // Under-staked: inflation increases, only apply max bound
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    // Over-staked: inflation decreases, only apply min bound
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'hours':
            // Show 24 hours
            const blocksPerHour = 720; // ~5 second blocks
            for (let hour = 0; hour <= 24; hour++) {
                labels.push(hour);
                rates.push(inflation * 100);

                // Calculate inflation after one hour
                for (let i = 0; i < blocksPerHour; i++) {
                    inflation += perBlockChange;
                }
                // Apply bounds check after the full period
                // Only apply the relevant bound based on direction
                if (perBlockChange > 0) {
                    // Under-staked: inflation increases, only apply max bound
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    // Over-staked: inflation decreases, only apply min bound
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'days':
            // Show 30 days
            const blocksPerDay = 17280; // ~5 second blocks
            for (let day = 0; day <= 30; day++) {
                labels.push(day);
                rates.push(inflation * 100);

                // Calculate inflation after one day
                for (let i = 0; i < blocksPerDay; i++) {
                    inflation += perBlockChange;
                }
                // Apply bounds check after the full period
                // Only apply the relevant bound based on direction
                if (perBlockChange > 0) {
                    // Under-staked: inflation increases, only apply max bound
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    // Over-staked: inflation decreases, only apply min bound
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'months':
            // Show 12 months
            const blocksPerMonth = 525960; // ~30.44 days per month
            for (let month = 0; month <= 12; month++) {
                labels.push(month);
                rates.push(inflation * 100);

                // Calculate inflation after one month
                for (let i = 0; i < blocksPerMonth; i++) {
                    inflation += perBlockChange;
                }
                // Apply bounds check after the full period
                // Only apply the relevant bound based on direction
                if (perBlockChange > 0) {
                    // Under-staked: inflation increases, only apply max bound
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    // Over-staked: inflation decreases, only apply min bound
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;
    }

    return { labels, rates };
}

// Switch chart period
function switchChartPeriod(period) {
    currentChartPeriod = period;

    // Update button states
    document.querySelectorAll('.chart-period-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-period="${period}"]`).classList.add('active');

    // Recalculate and update chart
    calculateRewards();
}

// Make functions globally available
window.switchChartPeriod = switchChartPeriod;

// Switch provision period
function switchProvisionPeriod(period) {
    currentProvisionPeriod = period;

    // Update button states
    document.querySelectorAll('.provision-period-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-period="${period}"]`).classList.add('active');

    // Recalculate and update provisions
    calculateRewards();
}

// Make functions globally available
window.switchProvisionPeriod = switchProvisionPeriod;

// Update chart statistics panel
function updateChartStats(rates) {
    const startRate = rates[0];
    const endRate = rates[rates.length - 1];
    const totalChange = endRate - startRate;

    document.getElementById('chart-start-rate').textContent = startRate.toFixed(2) + '%';
    document.getElementById('chart-end-rate').textContent = endRate.toFixed(2) + '%';
    document.getElementById('chart-total-change').textContent =
        (totalChange >= 0 ? '+' : '') + totalChange.toFixed(2) + '%';
}

// Utility function to format large numbers
function formatLargeNumber(num) {
    if (num >= 1e9) {
        return (num / 1e9).toFixed(1) + 'B';
    } else if (num >= 1e6) {
        return (num / 1e6).toFixed(1) + 'M';
    } else if (num >= 1e3) {
        return (num / 1e3).toFixed(1) + 'K';
    } else {
        return Math.round(num).toLocaleString();
    }
}

// Calculate time-based provisions
function calculateTimeBasedProvisions(provisions, params, period) {
    const blocksPerYear = params.blocksPerYear;
    let multiplier;
    
    switch (period) {
        case 'annual':
            multiplier = 1;
            break;
        case 'monthly':
            multiplier = 1 / 12;
            break;
        case 'daily':
            multiplier = 1 / 365;
            break;
        case 'hourly':
            multiplier = 1 / (365 * 24);
            break;
        default:
            multiplier = 1;
    }
    
    return {
        totalMinted: provisions.totalMinted * multiplier,
        communityPool: provisions.communityPool * multiplier,
        validatorRewards: provisions.validatorRewards * multiplier
    };
}

// Get provision labels based on period
function getProvisionLabels(period) {
    switch (period) {
        case 'annual':
            return {
                totalMinted: 'Total Minted Annually',
                communityPool: 'Community Pool (2%)',
                validatorRewards: 'Validator Rewards'
            };
        case 'monthly':
            return {
                totalMinted: 'Total Minted Monthly',
                communityPool: 'Community Pool Monthly',
                validatorRewards: 'Validator Rewards Monthly'
            };
        case 'daily':
            return {
                totalMinted: 'Total Minted Daily',
                communityPool: 'Community Pool Daily',
                validatorRewards: 'Validator Rewards Daily'
            };
        case 'hourly':
            return {
                totalMinted: 'Total Minted Hourly',
                communityPool: 'Community Pool Hourly',
                validatorRewards: 'Validator Rewards Hourly'
            };
        default:
            return {
                totalMinted: 'Total Minted Annually',
                communityPool: 'Community Pool (2%)',
                validatorRewards: 'Validator Rewards'
            };
    }
}

// Provisions Chart functionality
let provisionsChart = null;
let currentProvisionsChartPeriod = 'blocks';

// Generate provisions data for different time periods
function generateProvisionsData(params, startInflation, perBlockChange, period) {
    const labels = [];
    const totalMintedData = [];
    const communityPoolData = [];
    const validatorRewardsData = [];
    
    let inflation = startInflation;
    const communityTax = 0.02; // 2%

    switch (period) {
        case 'blocks':
            for (let block = 0; block <= 100000; block += 1000) {
                labels.push(block);
                
                // Calculate provisions at current inflation
                const totalMinted = inflation * params.totalSupply;
                const communityPool = totalMinted * communityTax;
                const validatorRewards = totalMinted - communityPool;
                
                totalMintedData.push(totalMinted / 1000000); // Convert to millions
                communityPoolData.push(communityPool / 1000000);
                validatorRewardsData.push(validatorRewards / 1000000);

                // Calculate inflation for next 1000 blocks
                for (let i = 0; i < 1000; i++) {
                    inflation += perBlockChange;
                }
                // Apply directional bounds check
                if (perBlockChange > 0) {
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'hours':
            const blocksPerHour = 720;
            for (let hour = 0; hour <= 24; hour++) {
                labels.push(hour);
                
                const totalMinted = inflation * params.totalSupply;
                const communityPool = totalMinted * communityTax;
                const validatorRewards = totalMinted - communityPool;
                
                totalMintedData.push(totalMinted / 1000000);
                communityPoolData.push(communityPool / 1000000);
                validatorRewardsData.push(validatorRewards / 1000000);

                for (let i = 0; i < blocksPerHour; i++) {
                    inflation += perBlockChange;
                }
                if (perBlockChange > 0) {
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'days':
            const blocksPerDay = 17280;
            for (let day = 0; day <= 30; day++) {
                labels.push(day);
                
                const totalMinted = inflation * params.totalSupply;
                const communityPool = totalMinted * communityTax;
                const validatorRewards = totalMinted - communityPool;
                
                totalMintedData.push(totalMinted / 1000000);
                communityPoolData.push(communityPool / 1000000);
                validatorRewardsData.push(validatorRewards / 1000000);

                for (let i = 0; i < blocksPerDay; i++) {
                    inflation += perBlockChange;
                }
                if (perBlockChange > 0) {
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;

        case 'months':
            const blocksPerMonth = 525960;
            for (let month = 0; month <= 12; month++) {
                labels.push(month);
                
                const totalMinted = inflation * params.totalSupply;
                const communityPool = totalMinted * communityTax;
                const validatorRewards = totalMinted - communityPool;
                
                totalMintedData.push(totalMinted / 1000000);
                communityPoolData.push(communityPool / 1000000);
                validatorRewardsData.push(validatorRewards / 1000000);

                for (let i = 0; i < blocksPerMonth; i++) {
                    inflation += perBlockChange;
                }
                if (perBlockChange > 0) {
                    inflation = Math.min(params.inflationMax, inflation);
                } else if (perBlockChange < 0) {
                    inflation = Math.max(params.inflationMin, inflation);
                }
            }
            break;
    }

    return { labels, totalMintedData, communityPoolData, validatorRewardsData };
}

// Create provisions chart
function createProvisionsChart() {
    const ctx = document.getElementById('provisionsChart');
    if (!ctx) return;

    provisionsChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'Total Minted (M)',
                    data: [],
                    borderColor: '#3042FB',
                    backgroundColor: 'rgba(48, 66, 251, 0.15)',
                    borderWidth: 4,
                    fill: true,
                    tension: 0.1,
                    pointRadius: 4,
                    pointHoverRadius: 6,
                    yAxisID: 'y'
                },
                {
                    label: 'Validator Rewards (M)',
                    data: [],
                    borderColor: '#F59E0B',
                    backgroundColor: 'rgba(245, 158, 11, 0.1)',
                    borderWidth: 3,
                    fill: false,
                    tension: 0.1,
                    pointRadius: 3,
                    pointHoverRadius: 5,
                    yAxisID: 'y'
                },
                {
                    label: 'Community Pool (M)',
                    data: [],
                    borderColor: '#10B981',
                    backgroundColor: 'rgba(16, 185, 129, 0.1)',
                    borderWidth: 3,
                    fill: false,
                    tension: 0.1,
                    pointRadius: 3,
                    pointHoverRadius: 5,
                    yAxisID: 'y1'
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false,
            },
            plugins: {
                title: {
                    display: true,
                    text: 'Token Provisions Over Time',
                    font: {
                        family: 'Inter',
                        size: 18,
                        weight: '600'
                    },
                    color: '#1F2937'
                },
                legend: {
                    display: true,
                    position: 'top',
                    labels: {
                        usePointStyle: true,
                        font: {
                            family: 'Inter',
                            size: 12
                        },
                        padding: 20
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(255, 255, 255, 0.98)',
                    titleColor: '#1F2937',
                    bodyColor: '#374151',
                    borderColor: '#E5E7EB',
                    borderWidth: 2,
                    cornerRadius: 12,
                    displayColors: true,
                    titleFont: {
                        family: 'Inter',
                        size: 14,
                        weight: '600'
                    },
                    bodyFont: {
                        family: 'Inter',
                        size: 13
                    },
                    callbacks: {
                        title: function(context) {
                            const period = currentProvisionsChartPeriod;
                            const labels = {
                                'blocks': 'Block',
                                'hours': 'Hour',
                                'days': 'Day',
                                'months': 'Month'
                            };
                            return `${labels[period]} ${context[0].label}`;
                        },
                        label: function(context) {
                            const value = context.parsed.y;
                            const label = context.dataset.label;
                            return `${label}: ${value.toFixed(2)}M tokens`;
                        }
                    }
                }
            },
            scales: {
                x: {
                    title: {
                        display: true,
                        text: 'Block Number (×1000)',
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#6B7280'
                    },
                    grid: {
                        color: 'rgba(107, 114, 128, 0.1)'
                    },
                    ticks: {
                        font: {
                            family: 'Inter',
                            size: 12
                        },
                        color: '#6B7280'
                    }
                },
                y: {
                    type: 'linear',
                    display: true,
                    position: 'left',
                    title: {
                        display: true,
                        text: 'Total Minted & Validator Rewards (M)',
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#6B7280'
                    },
                    grid: {
                        color: 'rgba(107, 114, 128, 0.1)'
                    },
                    ticks: {
                        font: {
                            family: 'Inter',
                            size: 12
                        },
                        color: '#6B7280',
                        callback: function(value) {
                            return value.toFixed(1) + 'M';
                        }
                    }
                },
                y1: {
                    type: 'linear',
                    display: true,
                    position: 'right',
                    title: {
                        display: true,
                        text: 'Community Pool (M)',
                        font: {
                            family: 'Inter',
                            size: 14,
                            weight: '600'
                        },
                        color: '#10B981'
                    },
                    grid: {
                        drawOnChartArea: false,
                    },
                    ticks: {
                        font: {
                            family: 'Inter',
                            size: 12
                        },
                        color: '#10B981',
                        callback: function(value) {
                            return value.toFixed(2) + 'M';
                        }
                    }
                }
            }
        }
    });
}

// Update provisions chart
function updateProvisionsChart(params) {
    if (!provisionsChart) {
        createProvisionsChart();
        // Continue to update with data after creation
    }

    const currentInflation = calculateCurrentInflation(params);
    const perBlockChange = calculatePerBlockChange(params);
    
    const data = generateProvisionsData(params, currentInflation, perBlockChange, currentProvisionsChartPeriod);
    
    // Update chart data
    provisionsChart.data.labels = data.labels;
    provisionsChart.data.datasets[0].data = data.totalMintedData;      // Total Minted
    provisionsChart.data.datasets[1].data = data.validatorRewardsData;  // Validator Rewards
    provisionsChart.data.datasets[2].data = data.communityPoolData;     // Community Pool
    
    // Update x-axis title based on period
    const xAxisTitle = {
        'blocks': 'Block Number (×1000)',
        'hours': 'Hours',
        'days': 'Days',
        'months': 'Months'
    };
    
    provisionsChart.options.scales.x.title.text = xAxisTitle[currentProvisionsChartPeriod];
    
    // Update chart with animation
    provisionsChart.update('active');
    
    // Update summary stats
    const startTotal = data.totalMintedData[0];
    const endTotal = data.totalMintedData[data.totalMintedData.length - 1];
    const totalChange = endTotal - startTotal;
    
    document.getElementById('provisions-start-total').textContent = startTotal.toFixed(1) + 'M';
    document.getElementById('provisions-end-total').textContent = endTotal.toFixed(1) + 'M';
    document.getElementById('provisions-total-change').textContent = 
        (totalChange >= 0 ? '+' : '') + totalChange.toFixed(1) + 'M';
}

// Switch provisions chart period
function switchProvisionsChartPeriod(period) {
    currentProvisionsChartPeriod = period;
    
    // Update button states
    document.querySelectorAll('.provisions-chart-period-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.getAttribute('data-period') === period) {
            btn.classList.add('active');
        }
    });
    
    // Update chart
    const params = getCurrentParameters();
    updateProvisionsChart(params);
}

// Get current parameters from form inputs
function getCurrentParameters() {
    return {
        inflationRateChange: parseFloat(document.getElementById('inflationRateChange').value) / 100,
        inflationMax: parseFloat(document.getElementById('inflationMax').value) / 100,
        inflationMin: parseFloat(document.getElementById('inflationMin').value) / 100,
        goalBonded: parseFloat(document.getElementById('goalBonded').value) / 100,
        initialInflation: parseFloat(document.getElementById('initialInflation').value) / 100,
        blocksPerYear: parseInt(document.getElementById('blocksPerYear').value),
        currentBonded: parseFloat(document.getElementById('currentBonded').value) / 100,
        totalSupply: parseInt(document.getElementById('totalSupply').value),
        validatorCommission: parseFloat(document.getElementById('validatorCommission').value) / 100
    };
}

// Make all functions globally available
window.calculateRewards = calculateRewards;
window.switchProvisionsChartPeriod = switchProvisionsChartPeriod;
window.getCurrentParameters = getCurrentParameters;