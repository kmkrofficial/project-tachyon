import React, { useState, useCallback } from 'react';
import { Plus, Trash2 } from 'lucide-react';

interface HostRule {
    domain: string;
    limit: number;
}

export const NetworkSettings: React.FC = () => {
    const [rules, setRules] = useState<HostRule[]>([]);
    const [newDomain, setNewDomain] = useState('');
    const [newLimit, setNewLimit] = useState(2);

    // Load existing rules on mount
    React.useEffect(() => {
        const loadRules = async () => {
            try {
                const saved = localStorage.getItem('tachyon-host-limits');
                if (saved) {
                    const parsed: HostRule[] = JSON.parse(saved);
                    setRules(parsed);
                    for (const rule of parsed) {
                        window.go?.app?.App?.SetHostLimit?.(rule.domain, rule.limit);
                    }
                }
            } catch { /* ignore */ }
        };
        loadRules();
    }, []);

    const persist = useCallback((updated: HostRule[]) => {
        setRules(updated);
        localStorage.setItem('tachyon-host-limits', JSON.stringify(updated));
    }, []);

    const addRule = useCallback(() => {
        const domain = newDomain.trim().toLowerCase();
        if (!domain || newLimit < 1) return;
        if (rules.some(r => r.domain === domain)) return;

        window.go?.app?.App?.SetHostLimit?.(domain, newLimit);
        persist([...rules, { domain, limit: newLimit }]);
        setNewDomain('');
        setNewLimit(2);
    }, [newDomain, newLimit, rules, persist]);

    const removeRule = useCallback((domain: string) => {
        window.go?.app?.App?.SetHostLimit?.(domain, 0);
        persist(rules.filter(r => r.domain !== domain));
    }, [rules, persist]);

    const updateLimit = useCallback((domain: string, limit: number) => {
        if (limit < 1) return;
        window.go?.app?.App?.SetHostLimit?.(domain, limit);
        persist(rules.map(r => r.domain === domain ? { ...r, limit } : r));
    }, [rules, persist]);

    return (
        <div className="space-y-6">
            <div className="bg-th-raised p-4 rounded-lg">
                <span className="block text-th-text font-medium mb-2">Concurrency Limits per Host</span>
                <p className="text-xs text-th-text-s mb-4">Limit simultaneous downloads from specific domains (e.g. mega.nz to 1).</p>

                {/* Existing Rules */}
                {rules.length > 0 && (
                    <div className="space-y-2 mb-4">
                        {rules.map(rule => (
                            <div key={rule.domain} className="flex items-center gap-3 bg-th-surface rounded-lg px-3 py-2">
                                <span className="flex-1 text-sm text-th-text font-mono">{rule.domain}</span>
                                <input
                                    type="number"
                                    min={1}
                                    max={32}
                                    value={rule.limit}
                                    onChange={e => updateLimit(rule.domain, parseInt(e.target.value) || 1)}
                                    className="w-16 bg-th-raised border border-th-border rounded px-2 py-1 text-sm text-th-text text-center"
                                />
                                <span className="text-xs text-th-text-m">max</span>
                                <button onClick={() => removeRule(rule.domain)} className="p-1 text-th-text-m hover:text-red-400 transition-colors">
                                    <Trash2 size={14} />
                                </button>
                            </div>
                        ))}
                    </div>
                )}

                {/* Add New Rule */}
                <div className="flex items-center gap-2">
                    <input
                        type="text"
                        placeholder="example.com"
                        value={newDomain}
                        onChange={e => setNewDomain(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && addRule()}
                        className="flex-1 bg-th-surface border border-th-border rounded-lg px-3 py-2 text-sm text-th-text placeholder-th-text-m"
                    />
                    <input
                        type="number"
                        min={1}
                        max={32}
                        value={newLimit}
                        onChange={e => setNewLimit(parseInt(e.target.value) || 1)}
                        className="w-16 bg-th-surface border border-th-border rounded-lg px-2 py-2 text-sm text-th-text text-center"
                    />
                    <button
                        onClick={addRule}
                        disabled={!newDomain.trim()}
                        className="p-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
                    >
                        <Plus size={16} />
                    </button>
                </div>
            </div>
        </div>
    );
};
