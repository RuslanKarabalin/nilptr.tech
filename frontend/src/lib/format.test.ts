import { describe, expect, it } from 'vitest';
import { formatDateLong, readingMinutes } from './format';

describe('formatDateLong', () => {
	it('formats in UTC without the locale', () => {
		expect(formatDateLong('2026-10-12T23:30:00Z')).toBe('Oct 12, 2026');
		expect(formatDateLong('2026-01-01T00:00:00Z')).toBe('Jan 1, 2026');
	});

	it('returns an empty string for missing or invalid dates', () => {
		expect(formatDateLong(null)).toBe('');
		expect(formatDateLong('nope')).toBe('');
	});
});

describe('readingMinutes', () => {
	it('rounds to whole minutes at 200 words per minute', () => {
		expect(readingMinutes(1000)).toBe(5);
		expect(readingMinutes(1099)).toBe(5);
		expect(readingMinutes(1100)).toBe(6);
	});

	it('is at least one minute', () => {
		expect(readingMinutes(0)).toBe(1);
		expect(readingMinutes(20)).toBe(1);
		expect(readingMinutes(Number.NaN)).toBe(1);
	});
});
