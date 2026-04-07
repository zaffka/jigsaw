import { render } from 'preact';
import { App } from './app';
import './index.css';

render(<App />, document.getElementById('app')!);

// Register service worker for PWA
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch((err) => {
      console.warn('SW registration failed:', err);
    });
  });
}
