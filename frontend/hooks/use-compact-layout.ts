'use client';

import { useSyncExternalStore } from 'react';

const query = '(width < 64rem)';
function subscribe(callback: () => void) {
  const media = window.matchMedia(query);
  media.addEventListener('change', callback);
  return () => media.removeEventListener('change', callback);
}
const getSnapshot = () => window.matchMedia(query).matches;
const getServerSnapshot = () => true;

export function useCompactLayout() {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
