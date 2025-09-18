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
    updateAnnualProvisionsDisplay(provisions);
    updateBlockProvisionsDisplay(blockProvisions, params);
    updateAPRDisplay(apr);
    updateCalculationDetails(params, currentInflation, perBlockChange, provisions, apr);

    // Update chart
    updateInflationChart(params);
}

// Calculate current inflation rate using actual Cosmos SDK NextInflationRate logic
function calculateCurrentInflation(params) {
    // Start with initial inflation from genesis (not a fixed base)
    let currentInflation = params.initialInflation;

    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases
        // Formula: current + (inflation_rate_change × (goal_bonded - current_bonded)) / blocks_per_year
        const gap = params.goalBonded - params.currentBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        currentInflation = Math.min(currentInflation + perBlockAdjustment, params.inflationMax);
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases
        // Formula: current - (inflation_rate_change × (current_bonded - goal_bonded)) / blocks_per_year
        const gap = params.currentBonded - params.goalBonded;
        const perBlockAdjustment = (params.inflationRateChange * gap) / params.blocksPerYear;
        currentInflation = Math.max(currentInflation - perBlockAdjustment, params.inflationMin);
    }

    return currentInflation;
}

// Calculate per-block change using correct Cosmos SDK formula
function calculatePerBlockChange(params) {
    if (params.currentBonded < params.goalBonded) {
        // Under-staked: inflation increases (positive change)
        // Formula: (inflationRateChange * gap) / blocksPerYear
        const gap = params.goalBonded - params.currentBonded;
    const perBlockChange = (params.inflationRateChange * gap) / params.blocksPerYear;
    return perBlockChange;
    } else if (params.currentBonded > params.goalBonded) {
        // Over-staked: inflation decreases (negative change)
        // Formula: -(inflationRateChange * gap) / blocksPerYear
        const gap = params.currentBonded - params.goalBonded;
        const perBlockChange = -(params.inflationRateChange * gap) / params.blocksPerYear;
        return perBlockChange;
    } else {
        // At target: no change
        return 0;
    }
}

// Calculate timeline projections
function calculateTimeline(params, currentInflation, perBlockChange) {
    return {
        block1: currentInflation,
        day1: currentInflation + (perBlockChange * 17280), // ~1 day of blocks
        week1: currentInflation + (perBlockChange * 120960), // ~1 week
        month1: currentInflation + (perBlockChange * 525960), // ~1 month
        year1: currentInflation + (perBlockChange * params.blocksPerYear)
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
    const baseAPR = provisions.validatorRewards / totalStaked;
    const delegatorAPR = baseAPR * (1 - params.validatorCommission);

    // Validators get delegator APR + commission from all their delegators
    // Assuming they have their own stake, they earn both rates
    const validatorAPR = baseAPR; // Full rate on their own stake

    // Commission is earned from the total rewards pool, not total staked
    const commissionEarnings = provisions.validatorRewards * params.validatorCommission;

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
    
    updateElement('current-bonded-calc', (params.currentBonded * 100).toFixed(0));
    updateElement('goal-bonded-calc', (params.goalBonded * 100).toFixed(0));
    updateElement('bonding-gap-calc', (gap * 100).toFixed(0));
    updateElement('rate-change-calc', (params.inflationRateChange * 100).toFixed(0));
    updateElement('gap-calc', (gap * 100).toFixed(0));

    // Calculate per-block adjustment correctly (reuse existing variable)
    // perBlockAdjustment is already calculated above with correct sign
    updateElement('adjustment-calc', (perBlockAdjustment * 100).toFixed(6));

    // Add blocks per year to display
    updateElement('blocks-per-year-calc', params.blocksPerYear.toLocaleString());
    document.getElementById('final-inflation-calc').textContent = (calculatedRate * 100).toFixed(1);
    document.getElementById('min-bound').textContent = (params.inflationMin * 100).toFixed(0) + '%';
    document.getElementById('max-bound').textContent = (params.inflationMax * 100).toFixed(0) + '%';
    document.getElementById('calculated-rate').textContent = (calculatedRate * 100).toFixed(1) + '%';
    document.getElementById('bounded-rate').textContent = (boundedRate * 100).toFixed(1) + '%';

    // Update formula direction using per-block adjustment
    const formulaElement = document.getElementById('adjustment-formula');
    if (params.currentBonded < params.goalBonded) {
        formulaElement.innerHTML = `Since Current Bonded &lt; Goal Bonded: <strong>Add</strong> per-block adjustment<br>
                                   Next Block Inflation = ${(baseRate * 100).toFixed(3)}% + ${(perBlockAdjustment * 100).toFixed(6)}% = <span id="final-inflation-calc">${(calculatedRate * 100).toFixed(6)}%</span>`;
    } else if (params.currentBonded > params.goalBonded) {
        formulaElement.innerHTML = `Since Current Bonded &gt; Goal Bonded: <strong>Subtract</strong> per-block adjustment<br>
                                   Next Block Inflation = ${(baseRate * 100).toFixed(3)}% - ${(perBlockAdjustment * 100).toFixed(6)}% = <span id="final-inflation-calc">${(calculatedRate * 100).toFixed(6)}%</span>`;
    } else {
        formulaElement.innerHTML = `Since Current Bonded = Goal Bonded: <strong>No</strong> adjustment<br>
                                   Next Block Inflation = ${(baseRate * 100).toFixed(3)}% + 0% = <span id="final-inflation-calc">${(baseRate * 100).toFixed(3)}%</span>`;
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

// Update annual provisions display
function updateAnnualProvisionsDisplay(provisions) {
    document.getElementById('totalMinted').textContent = formatLargeNumber(provisions.totalMinted);
    document.getElementById('communityPool').textContent = formatLargeNumber(provisions.communityPool);
    document.getElementById('validatorRewards').textContent = formatLargeNumber(provisions.validatorRewards);
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

    // Update calculation elements with safety checks
    updateElement('calc-goal', (params.goalBonded * 100).toFixed(0));
    updateElement('calc-current', (params.currentBonded * 100).toFixed(0));
    updateElement('calc-gap', gap.toFixed(0));
    updateElement('calc-goal-2', (params.goalBonded * 100).toFixed(0));
    updateElement('calc-gap-2', gap.toFixed(0));
    updateElement('calc-gap-ratio', gapRatio.toFixed(3));
    updateElement('calc-rate-change', (params.inflationRateChange * 100).toFixed(0));
    updateElement('calc-gap-ratio-2', gapRatio.toFixed(3));
    updateElement('calc-blocks', params.blocksPerYear.toLocaleString());
    // Update all per-block change displays
    const perBlockChangeValue = (perBlockChange * 100).toFixed(8);
    const perBlockChangeDisplay = perBlockChange >= 0 ? `+${perBlockChangeValue}%` : `${perBlockChangeValue}%`;
    
    // Update main per-block change display
    const perBlockElement = document.getElementById('perBlockChange');
    if (perBlockElement) {
        perBlockElement.textContent = perBlockChangeDisplay;
    }
    
    // Update calculation details
    document.getElementById('calc-per-block').textContent = perBlockChangeValue;
    document.getElementById('calc-direction').textContent = perBlockChangeValue;
    
    // Update the direction text dynamically
    const directionText = params.currentBonded < params.goalBonded ? 
        'Since Current Bonded < Goal Bonded: <strong>Inflation increases</strong>' :
        params.currentBonded > params.goalBonded ?
        'Since Current Bonded > Goal Bonded: <strong>Inflation decreases</strong>' :
        'Since Current Bonded = Goal Bonded: <strong>No change</strong>';
    
    // Update gap ratio calculation text (simplified to match Go code)
    const gapRatioText = `Gap = |Goal Bonded - Current Bonded| = |${(params.goalBonded * 100).toFixed(0)}% - ${(params.currentBonded * 100).toFixed(0)}%| = ${gap.toFixed(0)}%`;
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Calculate gap ratio')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                formulaElement.innerHTML = gapRatioText;
            }
        }
    });

    // Update per-block change calculation text (matching Go code exactly)
    const gapValue = Math.abs(params.goalBonded - params.currentBonded);
    const perBlockChangeText = `Per-Block Change = (Inflation Rate Change × Gap) / Blocks Per Year<br>= (${(params.inflationRateChange * 100).toFixed(0)}% × ${(gapValue * 100).toFixed(0)}%) / ${params.blocksPerYear.toLocaleString()}<br>= ${(perBlockChange * 100).toFixed(8)}% per block`;
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Calculate per-block change')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                formulaElement.innerHTML = perBlockChangeText;
            }
        }
    });

    // Find and update the direction step
    calculationSteps.forEach(step => {
        if (step.textContent.includes('Determine direction')) {
            const formulaElement = step.querySelector('.calculation-formula');
            if (formulaElement) {
                formulaElement.innerHTML = `${directionText} by <span id="calc-direction">${(perBlockChange * 100).toFixed(8)}%</span> per block`;
            }
        }
    });

    // Timeline calculation details
    document.getElementById('timeline-per-block').textContent = (perBlockChange * 100).toFixed(8) + '%';

    // Annual provisions calculation details
    document.getElementById('annual-inflation').textContent = (currentInflation * 100).toFixed(1) + '%';
    document.getElementById('annual-supply').textContent = params.totalSupply.toLocaleString();
    document.getElementById('annual-total').textContent = Math.round(provisions.totalMinted).toLocaleString();
    document.getElementById('annual-total-2').textContent = Math.round(provisions.totalMinted).toLocaleString();
    document.getElementById('annual-community').textContent = Math.round(provisions.communityPool).toLocaleString();
    document.getElementById('annual-total-3').textContent = Math.round(provisions.totalMinted).toLocaleString();
    document.getElementById('annual-community-2').textContent = Math.round(provisions.communityPool).toLocaleString();
    document.getElementById('annual-validators').textContent = Math.round(provisions.validatorRewards).toLocaleString();

    // Block provisions calculation details
    document.getElementById('block-annual').textContent = Math.round(provisions.validatorRewards).toLocaleString();
    document.getElementById('block-blocks').textContent = params.blocksPerYear.toLocaleString();
    document.getElementById('block-result').textContent = Math.round(provisions.validatorRewards / params.blocksPerYear).toLocaleString();
    document.getElementById('block-blocks-2').textContent = params.blocksPerYear.toLocaleString();
    document.getElementById('block-time-calc').textContent = (31536000 / params.blocksPerYear).toFixed(2);

    // APR calculation details
    document.getElementById('apr-supply').textContent = params.totalSupply.toLocaleString();
    document.getElementById('apr-bonded').textContent = (params.currentBonded * 100).toFixed(0) + '%';
    document.getElementById('apr-staked').textContent = Math.round(apr.totalStaked).toLocaleString();
    document.getElementById('apr-rewards').textContent = Math.round(provisions.validatorRewards).toLocaleString();
    document.getElementById('apr-staked-2').textContent = Math.round(apr.totalStaked).toLocaleString();
    document.getElementById('apr-base').textContent = (apr.baseAPR * 100).toFixed(2) + '%';
    document.getElementById('apr-base-2').textContent = (apr.baseAPR * 100).toFixed(2) + '%';
    document.getElementById('apr-commission').textContent = (params.validatorCommission * 100).toFixed(0) + '%';
    document.getElementById('apr-delegator-calc').textContent = (apr.delegatorAPR * 100).toFixed(2) + '%';
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
                    text: 'Inflation Rate vs Bonding Ratio',
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
                            return `Bonding Ratio: ${context[0].label}%`;
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

// Global variable for current chart period
let currentChartPeriod = 'blocks';

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
        blocks: 'Block Number',
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
            // Show first 1000 blocks
            for (let block = 0; block <= 1000; block += 10) {
                labels.push(block);
                rates.push(inflation * 100);

                // Calculate inflation for next 10 blocks
                for (let i = 0; i < 10; i++) {
                    inflation += perBlockChange;
                    inflation = Math.max(params.inflationMin, Math.min(params.inflationMax, inflation));
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
                    inflation = Math.max(params.inflationMin, Math.min(params.inflationMax, inflation));
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
                    inflation = Math.max(params.inflationMin, Math.min(params.inflationMax, inflation));
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

// Make all functions globally available
window.calculateRewards = calculateRewards;