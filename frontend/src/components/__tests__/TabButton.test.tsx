import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { TabButton } from '../TabButton';

// Provide a dummy icon component
const DummyIcon = (props: any) => <span data-testid="icon" />;

describe('TabButton', () => {
    it('renders label text', () => {
        render(<TabButton id="test" label="General" icon={DummyIcon as any} active={false} onClick={vi.fn()} />);
        expect(screen.getByText('General')).toBeInTheDocument();
    });

    it('renders the icon', () => {
        render(<TabButton id="test" label="General" icon={DummyIcon as any} active={false} onClick={vi.fn()} />);
        expect(screen.getByTestId('icon')).toBeInTheDocument();
    });

    it('calls onClick when clicked', () => {
        const onClick = vi.fn();
        render(<TabButton id="test" label="General" icon={DummyIcon as any} active={false} onClick={onClick} />);
        fireEvent.click(screen.getByText('General'));
        expect(onClick).toHaveBeenCalled();
    });

    it('applies active styles when active', () => {
        render(<TabButton id="test" label="General" icon={DummyIcon as any} active={true} onClick={vi.fn()} />);
        const button = screen.getByRole('button');
        expect(button.className).toContain('text-white');
    });

    it('applies inactive styles when not active', () => {
        render(<TabButton id="test" label="General" icon={DummyIcon as any} active={false} onClick={vi.fn()} />);
        const button = screen.getByRole('button');
        expect(button.className).toContain('text-gray-400');
    });
});
