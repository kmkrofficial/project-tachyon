import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ProgressBar } from '../ProgressBar';

describe('ProgressBar', () => {
    it('renders with correct width from progress', () => {
        const { container } = render(<ProgressBar progress={50} />);
        const bar = container.querySelector('.bg-th-accent');
        expect(bar).toHaveStyle({ width: '50%' });
    });

    it('clamps progress to 0-100 range', () => {
        const { container: c1 } = render(<ProgressBar progress={-10} />);
        expect(c1.querySelector('[style]')).toHaveStyle({ width: '0%' });

        const { container: c2 } = render(<ProgressBar progress={150} />);
        expect(c2.querySelector('[style]')).toHaveStyle({ width: '100%' });
    });

    it('uses green for completed status', () => {
        const { container } = render(<ProgressBar progress={100} status="completed" />);
        expect(container.querySelector('.bg-green-500')).toBeInTheDocument();
    });

    it('uses red for error status', () => {
        const { container } = render(<ProgressBar progress={30} status="error" />);
        expect(container.querySelector('.bg-red-500')).toBeInTheDocument();
    });

    it('uses yellow for paused status', () => {
        const { container } = render(<ProgressBar progress={60} status="paused" />);
        expect(container.querySelector('.bg-yellow-500')).toBeInTheDocument();
    });

    it('uses accent color for downloading (default) status', () => {
        const { container } = render(<ProgressBar progress={45} />);
        expect(container.querySelector('.bg-th-accent')).toBeInTheDocument();
    });
});
