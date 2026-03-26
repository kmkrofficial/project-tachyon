import { describe, it, expect } from 'vitest';
import { cn } from '../utils';

describe('cn (tailwind merge utility)', () => {
    it('merges class names', () => {
        expect(cn('foo', 'bar')).toBe('foo bar');
    });

    it('handles conditional classes', () => {
        expect(cn('base', false && 'hidden', 'visible')).toBe('base visible');
    });

    it('deduplicates tailwind classes', () => {
        // twMerge should resolve conflicts
        const result = cn('text-red-500', 'text-blue-500');
        expect(result).toBe('text-blue-500');
    });

    it('handles empty / undefined inputs', () => {
        expect(cn('', undefined, null, 'real')).toBe('real');
    });

    it('handles array inputs via clsx', () => {
        expect(cn(['a', 'b'])).toBe('a b');
    });
});
