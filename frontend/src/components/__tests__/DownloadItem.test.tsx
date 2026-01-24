import { render, screen, fireEvent } from "@testing-library/react";
import { DownloadItem } from "../DownloadItem";
import { describe, it, expect, vi } from "vitest";

const mockItem = {
    id: "test-id-123",
    filename: "test-video.mp4",
    progress: 45.5,
    speed_MBs: 12.5,
    eta: "2m 30s",
    status: "downloading" as const,
};

describe("DownloadItem", () => {
    it("renders filename and status correctly", () => {
        render(<DownloadItem item={mockItem} />);

        expect(screen.getByText("test-video.mp4")).toBeInTheDocument();
        expect(screen.getByText(/12.5 MB\/s/)).toBeInTheDocument();
    });

    it("renders progress bar with correct width", () => {
        const { container } = render(<DownloadItem item={mockItem} />);
        const progressBar = container.querySelector(".bg-blue-500");

        expect(progressBar).toHaveStyle({ width: "45.5%" });
    });

    it("shows error state color", () => {
        const errorItem = { ...mockItem, status: "error" as const, error: "Network Fail" };
        const { container } = render(<DownloadItem item={errorItem} />);

        expect(screen.getByText("Error")).toBeInTheDocument();
        expect(container.querySelector(".bg-red-500")).toBeInTheDocument();
    });
});
