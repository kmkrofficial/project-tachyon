import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DownloadItem } from '../../types';
import { DownloadDetailPanel } from '../DownloadDetailPanel';

const baseItem: DownloadItem = {
    id: 'test-id-1',
    url: 'http://example.com/file.zip',
    filename: 'file.zip',
    progress: 100,
    size: 10485760,
    status: 'completed',
    path: 'C:\\Downloads\\file.zip',
    category: 'compressed',
    accept_ranges: true,
    started_at: '2026-03-29T10:00:00Z',
    completed_at: '2026-03-29T10:01:30Z',
    elapsed: 90,
    avg_speed: 116508,
};

describe('DownloadDetailPanel', () => {
    it('renders source URL with copy button', () => {
        const onClose = vi.fn();
        render(<DownloadDetailPanel item={baseItem} onClose={onClose} />);
        expect(screen.getByText('http://example.com/file.zip')).toBeInTheDocument();
        expect(screen.getByTitle('Copy URL')).toBeInTheDocument();
    });

    it('shows file name', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('file.zip')).toBeInTheDocument();
    });

    it('shows formatted size', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('10.5 MB')).toBeInTheDocument();
    });

    it('shows category label', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('Archive')).toBeInTheDocument();
    });

    it('shows resume support as Yes', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('Yes (Range Supported)')).toBeInTheDocument();
    });

    it('shows resume support as No when false', () => {
        render(<DownloadDetailPanel item={{ ...baseItem, accept_ranges: false }} onClose={vi.fn()} />);
        expect(screen.getByText('No (Single Stream)')).toBeInTheDocument();
    });

    it('shows resume support as Unknown when undefined', () => {
        render(<DownloadDetailPanel item={{ ...baseItem, accept_ranges: undefined }} onClose={vi.fn()} />);
        expect(screen.getByText('Unknown')).toBeInTheDocument();
    });

    it('shows time taken', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('1m 30s')).toBeInTheDocument();
    });

    it('shows average speed', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('116.5 kB/s')).toBeInTheDocument();
    });

    it('shows save path', () => {
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        expect(screen.getByText('C:\\Downloads\\file.zip')).toBeInTheDocument();
    });

    it('calls onClose when X is clicked', () => {
        const onClose = vi.fn();
        render(<DownloadDetailPanel item={baseItem} onClose={onClose} />);
        // The X button is the only button with the close role in the header
        const header = screen.getByText('Download Details').parentElement!;
        fireEvent.click(header.querySelector('button')!);
        expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('copies URL to clipboard on copy button click', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, { clipboard: { writeText } });
        render(<DownloadDetailPanel item={baseItem} onClose={vi.fn()} />);
        fireEvent.click(screen.getByTitle('Copy URL'));
        expect(writeText).toHaveBeenCalledWith('http://example.com/file.zip');
    });

    it('shows dash for missing completed_at', () => {
        render(<DownloadDetailPanel item={{ ...baseItem, completed_at: undefined }} onClose={vi.fn()} />);
        const completedLabel = screen.getByText('Completed');
        const value = completedLabel.parentElement?.querySelector('.truncate');
        expect(value?.textContent).toBe('-');
    });
});
