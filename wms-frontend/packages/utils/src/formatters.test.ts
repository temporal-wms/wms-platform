import { describe, it, expect, beforeEach, vi } from 'vitest';
import {
  formatDate,
  formatDateTime,
  formatTime,
  formatRelativeTime,
  formatNumber,
  formatPercentage,
  formatCompactNumber,
  formatDuration,
  getOrderStatusColor,
  getWaveStatusColor,
  getWorkerStatusColor,
} from './formatters';

describe('Date Formatters', () => {
  describe('formatDate', () => {
    it('formats Date object', () => {
      const date = new Date('2026-01-15T10:30:00Z');
      const result = formatDate(date);
      expect(result).toMatch(/Jan 1[45], 2026/); // Account for timezone differences
    });

    it('formats date string', () => {
      const result = formatDate('2026-01-15T10:30:00Z');
      expect(result).toMatch(/Jan 1[45], 2026/);
    });

    it('handles different months', () => {
      const result = formatDate('2026-06-15T10:30:00Z');
      expect(result).toMatch(/Jun 1[45], 2026/);
    });
  });

  describe('formatDateTime', () => {
    it('formats Date object with time', () => {
      const date = new Date('2026-01-15T14:30:00Z');
      const result = formatDateTime(date);
      expect(result).toMatch(/Jan 1[45], 2026/);
      expect(result).toMatch(/\d{1,2}:\d{2}\s[AP]M/);
    });

    it('formats date string with time', () => {
      const result = formatDateTime('2026-01-15T14:30:00Z');
      expect(result).toMatch(/Jan 1[45], 2026/);
      expect(result).toMatch(/\d{1,2}:\d{2}\s[AP]M/);
    });
  });

  describe('formatTime', () => {
    it('formats time from Date object', () => {
      const date = new Date('2026-01-15T14:30:00Z');
      const result = formatTime(date);
      expect(result).toMatch(/\d{1,2}:\d{2}\s[AP]M/);
    });

    it('formats time from date string', () => {
      const result = formatTime('2026-01-15T14:30:00Z');
      expect(result).toMatch(/\d{1,2}:\d{2}\s[AP]M/);
    });
  });

  describe('formatRelativeTime', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-01-15T12:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns "Just now" for current time', () => {
      const now = new Date();
      expect(formatRelativeTime(now)).toBe('Just now');
    });

    it('returns "Just now" for < 1 minute ago', () => {
      const date = new Date(Date.now() - 30 * 1000); // 30 seconds ago
      expect(formatRelativeTime(date)).toBe('Just now');
    });

    it('returns minutes ago', () => {
      const date = new Date(Date.now() - 5 * 60 * 1000); // 5 minutes ago
      expect(formatRelativeTime(date)).toBe('5m ago');
    });

    it('returns hours ago', () => {
      const date = new Date(Date.now() - 3 * 60 * 60 * 1000); // 3 hours ago
      expect(formatRelativeTime(date)).toBe('3h ago');
    });

    it('returns days ago', () => {
      const date = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000); // 2 days ago
      expect(formatRelativeTime(date)).toBe('2d ago');
    });

    it('handles string dates', () => {
      const date = new Date(Date.now() - 10 * 60 * 1000).toISOString(); // 10 minutes ago
      expect(formatRelativeTime(date)).toBe('10m ago');
    });

    it('returns correct unit for edge cases', () => {
      // Exactly 60 seconds = 1 minute
      const oneMinute = new Date(Date.now() - 60 * 1000);
      expect(formatRelativeTime(oneMinute)).toBe('1m ago');

      // Exactly 60 minutes = 1 hour
      const oneHour = new Date(Date.now() - 60 * 60 * 1000);
      expect(formatRelativeTime(oneHour)).toBe('1h ago');

      // Exactly 24 hours = 1 day
      const oneDay = new Date(Date.now() - 24 * 60 * 60 * 1000);
      expect(formatRelativeTime(oneDay)).toBe('1d ago');
    });
  });
});

describe('Number Formatters', () => {
  describe('formatNumber', () => {
    it('formats numbers with thousand separators', () => {
      expect(formatNumber(1000)).toBe('1,000');
      expect(formatNumber(1000000)).toBe('1,000,000');
    });

    it('formats small numbers without separators', () => {
      expect(formatNumber(100)).toBe('100');
      expect(formatNumber(999)).toBe('999');
    });

    it('formats zero', () => {
      expect(formatNumber(0)).toBe('0');
    });

    it('formats negative numbers', () => {
      expect(formatNumber(-1000)).toBe('-1,000');
    });

    it('formats decimal numbers', () => {
      expect(formatNumber(1234.56)).toBe('1,234.56');
    });
  });

  describe('formatPercentage', () => {
    it('formats percentage with default 1 decimal', () => {
      expect(formatPercentage(85.5)).toBe('85.5%');
      expect(formatPercentage(100)).toBe('100.0%');
    });

    it('formats percentage with custom decimals', () => {
      expect(formatPercentage(85.5678, 2)).toBe('85.57%');
      expect(formatPercentage(85.5678, 0)).toBe('86%');
    });

    it('handles zero', () => {
      expect(formatPercentage(0)).toBe('0.0%');
    });

    it('handles negative percentages', () => {
      expect(formatPercentage(-10.5)).toBe('-10.5%');
    });
  });

  describe('formatCompactNumber', () => {
    it('formats thousands with K', () => {
      expect(formatCompactNumber(1000)).toBe('1K');
      expect(formatCompactNumber(1500)).toBe('1.5K');
      expect(formatCompactNumber(12500)).toBe('12.5K');
    });

    it('formats millions with M', () => {
      expect(formatCompactNumber(1000000)).toBe('1M');
      expect(formatCompactNumber(1500000)).toBe('1.5M');
    });

    it('formats billions with B', () => {
      expect(formatCompactNumber(1000000000)).toBe('1B');
      expect(formatCompactNumber(2500000000)).toBe('2.5B');
    });

    it('formats small numbers without suffix', () => {
      expect(formatCompactNumber(100)).toBe('100');
      expect(formatCompactNumber(999)).toBe('999');
    });

    it('limits to 1 decimal place', () => {
      expect(formatCompactNumber(1234)).toBe('1.2K');
      expect(formatCompactNumber(1567)).toBe('1.6K');
    });
  });
});

describe('Duration Formatters', () => {
  describe('formatDuration', () => {
    it('formats seconds only', () => {
      expect(formatDuration(30)).toBe('30s');
      expect(formatDuration(59)).toBe('59s');
    });

    it('formats minutes and seconds', () => {
      expect(formatDuration(90)).toBe('1m 30s');
      expect(formatDuration(125)).toBe('2m 5s');
    });

    it('formats hours and minutes', () => {
      expect(formatDuration(3600)).toBe('1h 0m');
      expect(formatDuration(3660)).toBe('1h 1m');
      expect(formatDuration(7325)).toBe('2h 2m');
    });

    it('handles zero', () => {
      expect(formatDuration(0)).toBe('0s');
    });

    it('handles edge cases', () => {
      expect(formatDuration(60)).toBe('1m 0s');
      expect(formatDuration(3599)).toBe('59m 59s');
    });

    it('floors fractional seconds', () => {
      expect(formatDuration(59.9)).toBe('59s');
      expect(formatDuration(90.5)).toBe('1m 30s');
    });
  });
});

describe('Status Color Mappers', () => {
  describe('getOrderStatusColor', () => {
    it('returns correct color for PENDING', () => {
      expect(getOrderStatusColor('PENDING')).toBe('bg-yellow-100 text-yellow-800');
    });

    it('returns correct color for VALIDATED', () => {
      expect(getOrderStatusColor('VALIDATED')).toBe('bg-blue-100 text-blue-800');
    });

    it('returns correct color for WAVED', () => {
      expect(getOrderStatusColor('WAVED')).toBe('bg-indigo-100 text-indigo-800');
    });

    it('returns correct color for PICKING', () => {
      expect(getOrderStatusColor('PICKING')).toBe('bg-purple-100 text-purple-800');
    });

    it('returns correct color for PICKED', () => {
      expect(getOrderStatusColor('PICKED')).toBe('bg-cyan-100 text-cyan-800');
    });

    it('returns correct color for PACKING', () => {
      expect(getOrderStatusColor('PACKING')).toBe('bg-teal-100 text-teal-800');
    });

    it('returns correct color for PACKED', () => {
      expect(getOrderStatusColor('PACKED')).toBe('bg-emerald-100 text-emerald-800');
    });

    it('returns correct color for SHIPPING', () => {
      expect(getOrderStatusColor('SHIPPING')).toBe('bg-lime-100 text-lime-800');
    });

    it('returns correct color for SHIPPED', () => {
      expect(getOrderStatusColor('SHIPPED')).toBe('bg-green-100 text-green-800');
    });

    it('returns correct color for COMPLETED', () => {
      expect(getOrderStatusColor('COMPLETED')).toBe('bg-green-100 text-green-800');
    });

    it('returns correct color for FAILED', () => {
      expect(getOrderStatusColor('FAILED')).toBe('bg-red-100 text-red-800');
    });

    it('returns correct color for DLQ', () => {
      expect(getOrderStatusColor('DLQ')).toBe('bg-red-200 text-red-900');
    });

    it('returns default color for unknown status', () => {
      expect(getOrderStatusColor('UNKNOWN')).toBe('bg-gray-100 text-gray-800');
      expect(getOrderStatusColor('')).toBe('bg-gray-100 text-gray-800');
    });
  });

  describe('getWaveStatusColor', () => {
    it('returns correct color for PLANNING', () => {
      expect(getWaveStatusColor('PLANNING')).toBe('bg-gray-100 text-gray-800');
    });

    it('returns correct color for READY', () => {
      expect(getWaveStatusColor('READY')).toBe('bg-blue-100 text-blue-800');
    });

    it('returns correct color for RELEASED', () => {
      expect(getWaveStatusColor('RELEASED')).toBe('bg-indigo-100 text-indigo-800');
    });

    it('returns correct color for IN_PROGRESS', () => {
      expect(getWaveStatusColor('IN_PROGRESS')).toBe('bg-yellow-100 text-yellow-800');
    });

    it('returns correct color for COMPLETED', () => {
      expect(getWaveStatusColor('COMPLETED')).toBe('bg-green-100 text-green-800');
    });

    it('returns correct color for CANCELLED', () => {
      expect(getWaveStatusColor('CANCELLED')).toBe('bg-red-100 text-red-800');
    });

    it('returns default color for unknown status', () => {
      expect(getWaveStatusColor('UNKNOWN')).toBe('bg-gray-100 text-gray-800');
      expect(getWaveStatusColor('')).toBe('bg-gray-100 text-gray-800');
    });
  });

  describe('getWorkerStatusColor', () => {
    it('returns correct color for AVAILABLE', () => {
      expect(getWorkerStatusColor('AVAILABLE')).toBe('bg-green-100 text-green-800');
    });

    it('returns correct color for BUSY', () => {
      expect(getWorkerStatusColor('BUSY')).toBe('bg-yellow-100 text-yellow-800');
    });

    it('returns correct color for BREAK', () => {
      expect(getWorkerStatusColor('BREAK')).toBe('bg-orange-100 text-orange-800');
    });

    it('returns correct color for OFFLINE', () => {
      expect(getWorkerStatusColor('OFFLINE')).toBe('bg-gray-100 text-gray-800');
    });

    it('returns default color for unknown status', () => {
      expect(getWorkerStatusColor('UNKNOWN')).toBe('bg-gray-100 text-gray-800');
      expect(getWorkerStatusColor('')).toBe('bg-gray-100 text-gray-800');
    });
  });
});
