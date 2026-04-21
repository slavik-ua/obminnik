import { useMarketStore } from '../store/useMarketStore';

const OrderBookLine = ({ price, vol, maxVol, type }: { price: number, vol: number, maxVol: number, type: 'bid' | 'ask' }) => {
  const percentage = (vol / maxVol) * 100;
  return (
    <div className="relative flex justify-between text-xs font-mono py-0.5 px-2 hover:bg-gray-800">
      <div 
        className={`absolute inset-0 opacity-20 ${type === 'bid' ? 'bg-green-500' : 'bg-red-500'}`}
        style={{ width: `${percentage}%`, left: type === 'ask' ? 'auto' : 0, right: type === 'ask' ? 0 : 'auto' }}
      />
      <span className={type === 'bid' ? 'text-green-400' : 'text-red-400'}>{price.toLocaleString()}</span>
      <span className="text-gray-300 z-10">{vol}</span>
    </div>
  );
};

export const OrderBook = () => {
  const { bids, asks } = useMarketStore();
  const maxVol = Math.max(...bids.map(b => b.total_vol), ...asks.map(a => a.total_vol), 1);

  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 w-64">
      <h3 className="text-gray-400 text-sm font-bold mb-2 uppercase tracking-wider">Order Book</h3>
      <div className="flex flex-col-reverse">
        {asks.slice(0, 15).reverse().map((ask) => (
          <OrderBookLine key={ask.price} price={ask.price} vol={ask.total_vol} maxVol={maxVol} type="ask" />
        ))}
      </div>
      <div className="py-2 text-center border-y border-gray-800 my-2 text-xl font-bold text-white">
        {(asks[0]?.price || bids[0]?.price || 0).toLocaleString()}
      </div>
      <div className="flex flex-col">
        {bids.slice(0, 15).map((bid) => (
          <OrderBookLine key={bid.price} price={bid.price} vol={bid.total_vol} maxVol={maxVol} type="bid" />
        ))}
      </div>
    </div>
  );
};