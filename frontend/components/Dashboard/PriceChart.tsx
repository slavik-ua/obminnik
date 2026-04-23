import React from 'react';
import { createChart, ColorType, IChartApi, ISeriesApi, Time, CandlestickData, CandlestickSeries } from 'lightweight-charts';
import { useMarketStore } from '../../store/useMarketStore';

export const PriceChart: React.FC = () => {
  const { priceHistory } = useMarketStore();
  const chartContainerRef = React.useRef<HTMLDivElement>(null);
  const chartRef = React.useRef<IChartApi | null>(null);
  const seriesRef = React.useRef<ISeriesApi<"Candlestick"> | null>(null);

  React.useEffect(() => {
    if (!chartContainerRef.current) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: 'rgba(255, 255, 255, 0.5)',
      },
      grid: {
        vertLines: { color: 'rgba(42, 46, 57, 0.1)' },
        horzLines: { color: 'rgba(42, 46, 57, 0.1)' },
      },
      crosshair: {
        mode: 0,
      },
      rightPriceScale: {
        borderColor: 'rgba(197, 203, 206, 0.1)',
        scaleMargins: {
          top: 0.2,
          bottom: 0.2,
        },
      },
      timeScale: {
        borderColor: 'rgba(197, 203, 206, 0.1)',
        timeVisible: true,
        secondsVisible: true,
      },
      handleScroll: {
        mouseWheel: true,
        pressedMouseMove: true,
      },
      handleScale: {
        axisPressedMouseMove: true,
        mouseWheel: true,
        pinch: true,
      },
    });

    const candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#00bf63',
      downColor: '#ef4444',
      borderVisible: false,
      wickUpColor: '#00bf63',
      wickDownColor: '#ef4444',
    });

    chartRef.current = chart;
    seriesRef.current = candlestickSeries;

    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ 
          width: chartContainerRef.current.clientWidth,
          height: chartContainerRef.current.clientHeight 
        });
      }
    };

    window.addEventListener('resize', handleResize);
    handleResize();

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.remove();
    };
  }, []);

  React.useEffect(() => {
    if (seriesRef.current && priceHistory.length > 0) {
      const data: CandlestickData<Time>[] = priceHistory.map(candle => ({
        time: candle.time as Time,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
      }));
      seriesRef.current.setData(data);
    }
  }, [priceHistory]);

  return (
    <div className="glass-card rounded-2xl p-6 min-h-[400px] lg:h-full shadow-2xl flex flex-col relative overflow-hidden group">
      <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 blur-3xl rounded-full -mr-16 -mt-16 pointer-events-none" />
      
      <div className="flex items-center justify-between mb-8 z-10">
        <h3 className="text-foreground font-black text-xs uppercase tracking-widest flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-[#00bf63] shadow-[0_0_8px_rgba(0,191,99,0.5)]" />
          Price History (Live Candlesticks)
        </h3>
        <div className="flex gap-3">
           <span className="px-2 py-0.5 rounded bg-[#00bf63]/10 text-[#00bf63] text-[9px] font-bold uppercase tracking-widest">Live</span>
           <span className="px-2 py-0.5 rounded bg-white/5 text-muted-foreground text-[9px] font-bold uppercase tracking-widest">1m</span>
        </div>
      </div>
      
      <div className="flex-1 w-full min-h-0 bg-background/5 rounded-xl border border-border/30 overflow-hidden relative">
        <div ref={chartContainerRef} className="absolute inset-0" />
      </div>
    </div>
  );
};
