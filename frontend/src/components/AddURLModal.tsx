import { X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

type AddURLModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onAdd: (url: string) => void;
};

export function AddURLModal({ isOpen, onClose, onAdd }: AddURLModalProps) {
    const [url, setUrl] = useState("");
    const inputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (isOpen) {
            setTimeout(() => inputRef.current?.focus(), 100);
        }
    }, [isOpen]);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (url.trim()) {
            onAdd(url.trim());
            setUrl("");
            onClose();
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
            <div className="bg-gray-900 border border-gray-700 rounded-xl shadow-2xl w-full max-w-lg p-6 animate-in fade-in zoom-in duration-200">
                <div className="flex justify-between items-center mb-6">
                    <h2 className="text-xl font-semibold text-white">Add New Download</h2>
                    <button onClick={onClose} className="text-gray-400 hover:text-white">
                        <X size={20} />
                    </button>
                </div>

                <form onSubmit={handleSubmit}>
                    <input
                        ref={inputRef}
                        type="text"
                        placeholder="Enter URL here..."
                        className="w-full bg-gray-800 border-gray-700 text-white rounded-lg px-4 py-3 focus:ring-2 focus:ring-blue-500 focus:outline-none mb-6"
                        value={url}
                        onChange={(e) => setUrl(e.target.value)}
                    />

                    <div className="flex justify-end gap-3">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 text-gray-300 hover:text-white"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={!url.trim()}
                            className="bg-blue-600 hover:bg-blue-500 text-white px-6 py-2 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            Start Download
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}
