/**
 * Highcharts Theme Configurations
 * Light and dark themes for professional financial charts
 */

import type { Options } from 'highcharts'

/**
 * Light theme configuration for Highcharts
 */
export const lightTheme: Partial<Options> = {
  colors: [
    '#2caffe', // Highcharts standard blue
    '#544fc5', // Purple
    '#00e272', // Green
    '#fe6a35', // Orange
    '#6b8abc', // Gray blue
    '#d568fb', // Pink
    '#2ee0ca', // Cyan
    '#fa4b42', // Red
    '#feb56a', // Light orange
    '#91e8e1', // Light cyan
  ],
  
  chart: {
    backgroundColor: 'transparent',
    style: {
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      color: '#333333'
    }
  },
  
  title: {
    style: {
      color: '#333333',
      fontSize: '16px',
      fontWeight: 'bold'
    }
  },
  
  subtitle: {
    style: {
      color: '#666666'
    }
  },
  
  xAxis: {
    gridLineColor: '#e6e6e6',
    gridLineWidth: 1,
    lineColor: '#cccccc',
    tickColor: '#cccccc',
    labels: {
      style: {
        color: '#666666'
      }
    },
    title: {
      style: {
        color: '#333333'
      }
    }
  },
  
  yAxis: {
    gridLineColor: '#e6e6e6',
    gridLineWidth: 1,
    lineColor: '#cccccc',
    tickColor: '#cccccc',
    labels: {
      style: {
        color: '#666666'
      }
    },
    title: {
      style: {
        color: '#333333'
      }
    }
  },
  
  tooltip: {
    backgroundColor: 'rgba(255, 255, 255, 0.95)',
    borderColor: '#cccccc',
    style: {
      color: '#333333'
    }
  },
  
  plotOptions: {
    candlestick: {
      lineColor: '#FF6F6F', // Matches Highcharts demo
      upLineColor: '#6FB76F', // Matches Highcharts demo
      color: '#FF6F6F', // Down color - matches demo
      upColor: '#6FB76F' // Up color - matches demo
    },
    ohlc: {
      color: '#FF6F6F',
      upColor: '#6FB76F'
    }
  },
  
  legend: {
    backgroundColor: 'rgba(255, 255, 255, 0.9)',
    borderColor: '#cccccc',
    itemStyle: {
      color: '#333333'
    },
    itemHoverStyle: {
      color: '#000000'
    },
    itemHiddenStyle: {
      color: '#cccccc'
    }
  },
  
  credits: {
    enabled: false
  },
  
  navigator: {
    maskFill: 'rgba(102, 133, 194, 0.3)',
    series: {
      color: '#5679c4',
      lineColor: '#5679c4'
    },
    xAxis: {
      gridLineColor: '#e6e6e6',
      labels: {
        style: {
          color: '#666666'
        }
      }
    }
  },
  
  rangeSelector: {
    buttonTheme: {
      fill: '#f7f7f7',
      stroke: '#cccccc',
      style: {
        color: '#333333'
      },
      states: {
        hover: {
          fill: '#e6e6e6',
          stroke: '#333333',
          style: {
            color: '#333333'
          }
        },
        select: {
          fill: '#0066cc',
          stroke: '#0066cc',
          style: {
            color: '#ffffff'
          }
        }
      }
    },
    inputBoxBorderColor: '#cccccc',
    inputStyle: {
      backgroundColor: '#ffffff',
      color: '#333333'
    },
    labelStyle: {
      color: '#666666'
    }
  },
  
  scrollbar: {
    barBackgroundColor: '#cccccc',
    barBorderColor: '#cccccc',
    buttonArrowColor: '#666666',
    buttonBackgroundColor: '#e6e6e6',
    buttonBorderColor: '#cccccc',
    rifleColor: '#666666',
    trackBackgroundColor: '#f2f2f2',
    trackBorderColor: '#f2f2f2'
  },
  
  navigation: {
    buttonOptions: {
      theme: {
        fill: '#f7f7f7',
        stroke: '#cccccc',
        'stroke-width': 1,
        r: 3,
        style: {
          color: '#333333',
          fontWeight: 'bold'
        },
        states: {
          hover: {
            fill: '#e6e6e6',
            stroke: '#333333',
            style: {
              color: '#333333'
            }
          },
          select: {
            fill: '#e6f2ff',
            stroke: '#2f7ed8',
            style: {
              color: '#2f7ed8'
            }
          }
        }
      },
      symbolFill: '#333333',
      symbolStroke: '#333333',
      symbolStrokeWidth: 2
    },
    menuStyle: {
      background: '#ffffff',
      border: '1px solid #cccccc',
      borderRadius: '4px',
      padding: '5px',
      boxShadow: '0 2px 5px rgba(0,0,0,0.15)'
    },
    menuItemStyle: {
      color: '#333333',
      fontSize: '13px',
      padding: '5px 10px',
      cursor: 'pointer'
    },
    menuItemHoverStyle: {
      background: '#f0f0f0',
      color: '#000000'
    }
  }
}

/**
 * Dark theme configuration for Highcharts
 */
export const darkTheme: Partial<Options> = {
  colors: [
    '#2caffe', // Highcharts standard blue (same in dark)
    '#544fc5', // Purple
    '#00e272', // Green
    '#fe6a35', // Orange
    '#6b8abc', // Gray blue
    '#d568fb', // Pink
    '#2ee0ca', // Cyan
    '#fa4b42', // Red
    '#feb56a', // Light orange
    '#91e8e1', // Light cyan
  ],
  
  chart: {
    backgroundColor: 'transparent',
    style: {
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
      color: '#e0e0e0'
    }
  },
  
  title: {
    style: {
      color: '#e0e0e0',
      fontSize: '16px',
      fontWeight: 'bold'
    }
  },
  
  subtitle: {
    style: {
      color: '#a0a0a0'
    }
  },
  
  xAxis: {
    gridLineColor: '#404040',
    gridLineWidth: 1,
    lineColor: '#505050',
    tickColor: '#505050',
    labels: {
      style: {
        color: '#a0a0a0'
      }
    },
    title: {
      style: {
        color: '#e0e0e0'
      }
    }
  },
  
  yAxis: {
    gridLineColor: '#404040',
    gridLineWidth: 1,
    lineColor: '#505050',
    tickColor: '#505050',
    labels: {
      style: {
        color: '#a0a0a0'
      }
    },
    title: {
      style: {
        color: '#e0e0e0'
      }
    }
  },
  
  tooltip: {
    backgroundColor: 'rgba(30, 30, 30, 0.95)',
    borderColor: '#505050',
    style: {
      color: '#e0e0e0'
    }
  },
  
  plotOptions: {
    candlestick: {
      lineColor: '#FF6F6F', // Keep consistent with light theme
      upLineColor: '#6FB76F', // Keep consistent with light theme
      color: '#FF6F6F', // Down color
      upColor: '#6FB76F' // Up color
    },
    ohlc: {
      color: '#f87171',
      upColor: '#4ade80'
    }
  },
  
  legend: {
    backgroundColor: 'rgba(30, 30, 30, 0.9)',
    borderColor: '#505050',
    itemStyle: {
      color: '#e0e0e0'
    },
    itemHoverStyle: {
      color: '#ffffff'
    },
    itemHiddenStyle: {
      color: '#606060'
    }
  },
  
  credits: {
    enabled: false
  },
  
  navigator: {
    maskFill: 'rgba(96, 165, 250, 0.3)',
    series: {
      color: '#60a5fa',
      lineColor: '#60a5fa'
    },
    xAxis: {
      gridLineColor: '#404040',
      labels: {
        style: {
          color: '#a0a0a0'
        }
      }
    }
  },
  
  rangeSelector: {
    buttonTheme: {
      fill: '#2a2a2a',
      stroke: '#505050',
      style: {
        color: '#e0e0e0'
      },
      states: {
        hover: {
          fill: '#3a3a3a',
          stroke: '#606060',
          style: {
            color: '#ffffff'
          }
        },
        select: {
          fill: '#0066cc',
          stroke: '#0066cc',
          style: {
            color: '#ffffff'
          }
        }
      }
    },
    inputBoxBorderColor: '#505050',
    inputStyle: {
      backgroundColor: '#2a2a2a',
      color: '#e0e0e0'
    },
    labelStyle: {
      color: '#a0a0a0'
    }
  },
  
  scrollbar: {
    barBackgroundColor: '#505050',
    barBorderColor: '#505050',
    buttonArrowColor: '#a0a0a0',
    buttonBackgroundColor: '#3a3a3a',
    buttonBorderColor: '#505050',
    rifleColor: '#a0a0a0',
    trackBackgroundColor: '#2a2a2a',
    trackBorderColor: '#2a2a2a'
  },
  
  navigation: {
    buttonOptions: {
      theme: {
        fill: '#2a2a2a',
        stroke: '#505050',
        'stroke-width': 1,
        r: 3,
        style: {
          color: '#e0e0e0',
          fontWeight: 'bold'
        },
        states: {
          hover: {
            fill: '#3a3a3a',
            stroke: '#606060',
            style: {
              color: '#ffffff'
            }
          },
          select: {
            fill: '#1a3d60',
            stroke: '#2f7ed8',
            style: {
              color: '#60a5fa'
            }
          }
        }
      },
      symbolFill: '#e0e0e0',
      symbolStroke: '#e0e0e0',
      symbolStrokeWidth: 2
    },
    menuStyle: {
      background: '#2a2a2a',
      border: '1px solid #505050',
      borderRadius: '4px',
      padding: '5px',
      boxShadow: '0 2px 5px rgba(0,0,0,0.3)'
    },
    menuItemStyle: {
      color: '#e0e0e0',
      fontSize: '13px',
      padding: '5px 10px',
      cursor: 'pointer'
    },
    menuItemHoverStyle: {
      background: '#3a3a3a',
      color: '#ffffff'
    }
  }
}

/**
 * Apply theme to Highcharts instance
 */
export function applyHighchartsTheme(Highcharts: any, theme: 'light' | 'dark') {
  const themeOptions = theme === 'dark' ? darkTheme : lightTheme
  
  // Apply the theme globally
  Highcharts.setOptions(themeOptions)
  
  // Also apply stock tools specific styling
  Highcharts.setOptions({
    stockTools: {
      gui: {
        // iconsURL removed - let Highcharts use default built-in icons
        buttons: {
          style: {
            fill: theme === 'dark' ? '#cbd5e0' : '#4a5568',
            stroke: theme === 'dark' ? '#cbd5e0' : '#4a5568'
          }
        }
      }
    }
  })
}