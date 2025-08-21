/**
 * Highcharts Module Loader
 * Professional singleton pattern for loading and initializing all Highcharts modules
 * Ensures modules are loaded once and in the correct order for Next.js static export
 * Supports dynamic theme switching between light and dark modes
 */

// Type imports for TypeScript support
import type { Options } from 'highcharts'
import { applyHighchartsTheme, lightTheme, darkTheme } from './highcharts-themes'

// Singleton instance
let highchartsInstance: any = null
let loadingPromise: Promise<any> | null = null

/**
 * Load and initialize all Highcharts modules
 * Uses dynamic imports for code splitting and proper Next.js compatibility
 */
export async function loadHighchartsModules(): Promise<any> {
  // Return existing instance if already loaded
  if (highchartsInstance) {
    return highchartsInstance
  }

  // Return existing loading promise if already loading
  if (loadingPromise) {
    return loadingPromise
  }

  // Start loading process
  loadingPromise = loadModules()
  
  try {
    highchartsInstance = await loadingPromise
    return highchartsInstance
  } finally {
    loadingPromise = null
  }
}

/**
 * Internal function to load all modules
 */
async function loadModules() {
  try {
    // Dynamic import of Highcharts Stock (base module)
    const Highcharts = (await import('highcharts/highstock')).default

    // Only proceed if Highcharts loaded as an object (not a function)
    if (typeof Highcharts !== 'object') {
      throw new Error('Highcharts did not load correctly')
    }

    // Load and initialize modules - Highcharts modules export factory functions directly
    console.log('📊 Loading Highcharts modules...')
    
    // CRITICAL: Load indicators module first - required for all technical indicators
    // The indicators-all module includes ALL indicator types
    try {
      const indicatorsModule = await import('highcharts/indicators/indicators-all')
      const initIndicators = indicatorsModule.default || indicatorsModule
      
      if (typeof initIndicators === 'function') {
        initIndicators(Highcharts)
        console.log('✅ Loaded indicators-all module')
        
        // Verify indicators are available
        if (Highcharts.seriesTypes && Object.keys(Highcharts.seriesTypes).length > 10) {
          console.log('✅ Indicators loaded successfully, available types:', Object.keys(Highcharts.seriesTypes).filter(t => t.includes('sma') || t.includes('ema') || t.includes('rsi')))
        }
      } else {
        console.warn('⚠️ Indicators module format unexpected, attempting fallback')
        // Try to load individual indicator modules as fallback
        const fallbackIndicators = [
          'highcharts/indicators/indicators',
          'highcharts/indicators/ema',
          'highcharts/indicators/sma',
          'highcharts/indicators/rsi',
          'highcharts/indicators/macd',
          'highcharts/indicators/bollinger-bands'
        ]
        
        for (const modPath of fallbackIndicators) {
          try {
            const mod = await import(modPath)
            const init = mod.default || mod
            if (typeof init === 'function') {
              init(Highcharts)
              console.log(`✅ Loaded ${modPath.split('/').pop()}`)
            }
          } catch (e) {
            console.warn(`Failed to load ${modPath}`)
          }
        }
      }
    } catch (error) {
      console.error('❌ Failed to load indicators module:', error)
    }

    // Load other required modules in the correct order
    // IMPORTANT: exporting must be loaded before full-screen (dependency)
    const modules = [
      { name: 'drag-panes', loader: () => import('highcharts/modules/drag-panes') },
      { name: 'annotations-advanced', loader: () => import('highcharts/modules/annotations-advanced') },
      { name: 'price-indicator', loader: () => import('highcharts/modules/price-indicator') },
      { name: 'exporting', loader: () => import('highcharts/modules/exporting') },  // Must be before full-screen
      { name: 'full-screen', loader: () => import('highcharts/modules/full-screen') },  // Depends on exporting
      { name: 'stock-tools', loader: () => import('highcharts/modules/stock-tools') },
      { name: 'heikinashi', loader: () => import('highcharts/modules/heikinashi') },
      { name: 'hollowcandlestick', loader: () => import('highcharts/modules/hollowcandlestick') },
      { name: 'export-data', loader: () => import('highcharts/modules/export-data') },
      { name: 'accessibility', loader: () => import('highcharts/modules/accessibility') }
    ]

    // Load each module and initialize it
    for (const { name, loader } of modules) {
      try {
        const module = await loader()
        let initFunc = null
        
        // Try multiple patterns to find the initialization function
        // Pattern 1: Direct function export (CommonJS style)
        if (typeof module === 'function') {
          initFunc = module
        }
        // Pattern 2: Default export
        else if (module && module.default && typeof module.default === 'function') {
          initFunc = module.default
        }
        // Pattern 3: Named export matching module name
        else if (module && module[name] && typeof module[name] === 'function') {
          initFunc = module[name]
        }
        // Pattern 4: Check for any function in the module
        else if (module && typeof module === 'object') {
          const keys = Object.keys(module)
          for (const key of keys) {
            if (typeof module[key] === 'function') {
              console.log(`${name}: Found function at key: ${key}`)
              initFunc = module[key]
              break
            }
          }
        }
        
        if (initFunc) {
          initFunc(Highcharts)
          console.log(`✅ Loaded ${name} module`)
        } else {
          console.error(`❌ Failed to initialize ${name} module - unexpected export format:`, module)
        }
      } catch (err) {
        console.error(`❌ Failed to load ${name} module:`, err)
      }
    }

    // Set language options (theme-independent)
    Highcharts.setOptions({
      lang: {
        thousandsSep: ',',
        decimalPoint: '.',
        rangeSelectorZoom: 'Period:',
        months: ['January', 'February', 'March', 'April', 'May', 'June', 
                 'July', 'August', 'September', 'October', 'November', 'December'],
        weekdays: ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
        shortMonths: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 
                      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
      },
      credits: {
        enabled: false
      }
    })
    
    // Theme will be applied dynamically by the component based on current theme

    console.log('✅ Highcharts modules loaded successfully')
    return Highcharts

  } catch (error) {
    console.error('❌ Failed to load Highcharts modules:', error)
    throw error
  }
}

/**
 * Create chart options with proper typing
 */
export function createChartOptions(
  ticker: string,
  chartData: any,
  resistanceZone: any,
  fibonacciPoints: any,
  theme: 'light' | 'dark' = 'light'
): Options {
  return {
    chart: {
      // Don't set fixed height - let container determine it
      backgroundColor: 'transparent',
      events: {
        // Force reflow when entering/exiting fullscreen
        fullscreenOpen: function() {
          // Use the theme passed as parameter (single source of truth)
          const themeColors = theme === 'dark' ? darkTheme : lightTheme
          const backgroundColor = theme === 'dark' ? '#1a202c' : '#ffffff'
          
          // Apply complete theme update for fullscreen (not just background)
          this.update({
            chart: {
              backgroundColor: backgroundColor,
              style: themeColors.chart?.style
            },
            title: {
              style: themeColors.title?.style
            },
            xAxis: {
              labels: { style: themeColors.xAxis?.labels?.style },
              gridLineColor: themeColors.xAxis?.gridLineColor,
              lineColor: themeColors.xAxis?.lineColor,
              tickColor: themeColors.xAxis?.tickColor
            },
            yAxis: [{
              labels: { style: themeColors.yAxis?.labels?.style },
              gridLineColor: themeColors.yAxis?.gridLineColor,
              lineColor: themeColors.yAxis?.lineColor,
              tickColor: themeColors.yAxis?.tickColor,
              title: { style: themeColors.yAxis?.title?.style }
            }, {
              labels: { style: themeColors.yAxis?.labels?.style },
              gridLineColor: themeColors.yAxis?.gridLineColor,
              lineColor: themeColors.yAxis?.lineColor,
              tickColor: themeColors.yAxis?.tickColor,
              title: { style: themeColors.yAxis?.title?.style }
            }],
            tooltip: {
              backgroundColor: themeColors.tooltip?.backgroundColor,
              borderColor: themeColors.tooltip?.borderColor,
              style: themeColors.tooltip?.style
            },
            rangeSelector: themeColors.rangeSelector,
            navigator: themeColors.navigator,
            scrollbar: themeColors.scrollbar
          }, false)
          
          this.reflow()
        },
        fullscreenClose: function() {
          // Restore transparent background when exiting fullscreen
          this.update({
            chart: {
              backgroundColor: 'transparent'
            }
          }, false)
          
          this.reflow()
        }
      }
    },
    
    time: {
      useUTC: false
    },
    
    title: {
      text: `${ticker} - Technical Analysis`,
      style: {
        fontSize: '16px',
        fontWeight: 'bold'
      }
    },
    
    rangeSelector: {
      buttons: [{
        type: 'day',
        count: 1,
        text: '1D'
      }, {
        type: 'week',
        count: 1,
        text: '1W'
      }, {
        type: 'month',
        count: 1,
        text: '1M'
      }, {
        type: 'month',
        count: 3,
        text: '3M'
      }, {
        type: 'month',
        count: 6,
        text: '6M'
      }, {
        type: 'ytd',
        text: 'YTD'
      }, {
        type: 'year',
        count: 1,
        text: '1Y'
      }, {
        type: 'all',
        text: 'All'
      }],
      selected: 2,
      inputEnabled: true
    },
    
    yAxis: [{
      labels: {
        align: 'left',
        x: 2
      },
      height: '75%',  // Increased from 60% since no RSI panel
      lineWidth: 2,
      resize: {
        enabled: true
      },
      title: {
        text: 'Price (IQD)'
      },
      plotBands: resistanceZone.from > 0 ? [{
        color: 'rgba(169, 255, 101, 0.4)',
        from: resistanceZone.from,
        to: resistanceZone.to,
        zIndex: 3,
        label: {
          text: 'Resistance Zone',
          style: {
            color: '#606060',
            fontWeight: 'bold'
          }
        }
      }] : []
    }, {
      labels: {
        align: 'left',
        x: 2
      },
      top: '75%',  // Adjusted from 60%
      height: '25%',  // Increased from 20%
      offset: 0,
      lineWidth: 2,
      title: {
        text: 'Volume'
      }
    }],
    
    tooltip: {
      shape: 'square',
      headerShape: 'callout',
      borderWidth: 0,
      shadow: false,
      split: true,
      shared: false,
      valueDecimals: 2,  // Always show 2 decimal places
      pointFormatter: function() {
        // For OHLC series, format all values with 2 decimals
        if (this.series.type === 'candlestick' || this.series.type === 'ohlc') {
          return '<b>' + this.series.name + '</b><br/>' +
            'Open: ' + this.open.toFixed(2) + '<br/>' +
            'High: ' + this.high.toFixed(2) + '<br/>' +
            'Low: ' + this.low.toFixed(2) + '<br/>' +
            'Close: ' + this.close.toFixed(2) + '<br/>'
        }
        // For other series types
        return '<span style="color:' + this.color + '">●</span> ' +
               this.series.name + ': <b>' + (typeof this.y === 'number' ? this.y.toFixed(2) : this.y) + '</b><br/>'
      }
    },
    
    stockTools: {
      gui: {
        enabled: true,
        buttons: [
          'indicators',
          'separator',
          'simpleShapes',
          'lines',
          'crookedLines',
          'measure',
          'advanced',
          'toggleAnnotations',
          'separator',
          'verticalLabels',
          'flags',
          'separator',
          'zoomChange',
          'fullScreen',
          'typeChange',
          'separator',
          'currentPriceIndicator',
          'saveChart'
        ],
        definitions: {
          typeChange: {
            items: ['typeCandlestick', 'typeOHLC', 'typeLine', 'typeHeikinAshi', 'typeHollowCandlestick']
          }
        }
      }
    },
    
    navigation: {
      annotationsOptions: {
        shapeOptions: {
          fill: 'rgba(255, 0, 0, 0.2)',
          stroke: 'rgba(255, 0, 0, 1)',
          strokeWidth: 2
        }
      }
    },
    
    annotations: [], // No default annotations - users can add via stock tools
    
    series: [{
      type: 'candlestick',
      id: ticker, // Simplified ID for better indicator compatibility
      name: ticker,
      data: chartData.ohlc,
      yAxis: 0,
      color: '#FF6F6F',
      upColor: '#6FB76F',
      lineColor: '#FF6F6F',
      upLineColor: '#6FB76F',
      dataGrouping: {
        enabled: false
      },
      // Required for indicators to work properly
      navigatorOptions: {
        enabled: true
      },
      showInNavigator: true
    }, {
      type: 'column',
      id: 'volume',
      name: 'Volume',
      data: chartData.volume,
      yAxis: 1,
      color: 'rgba(100, 100, 100, 0.5)',
      dataGrouping: {
        enabled: false
      },
      showInNavigator: false
    }
    // No default indicators - users can add any indicators via stock tools GUI
    ],
    
    responsive: {
      rules: [{
        condition: {
          maxWidth: 800
        },
        chartOptions: {
          chart: {
            height: 400
          },
          rangeSelector: {
            inputEnabled: false
          },
          stockTools: {
            gui: {
              enabled: false
            }
          }
        }
      }]
    },
    
    exporting: {
      enabled: true,
      buttons: {
        contextButton: {
          menuItems: [
            'viewFullscreen',  // Full-screen mode (requires full-screen module)
            'separator',
            'downloadPNG',
            'downloadJPEG',
            'downloadPDF',
            'downloadSVG',
            'separator',
            'downloadCSV',
            'downloadXLS'
          ]
        }
      }
    }
  } as Options
}

/**
 * Reset the loader (useful for testing or hot reload)
 */
export function resetHighchartsLoader() {
  highchartsInstance = null
  loadingPromise = null
}