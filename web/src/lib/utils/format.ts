/**
 * Format a number with K/M suffix for display.
 */
export function formatCount(count: number): string {
	if (count >= 1_000_000) return `${(count / 1_000_000).toFixed(1)}M`;
	if (count >= 1_000) return `${(count / 1_000).toFixed(1)}K`;
	return String(count);
}

/**
 * Format download velocity with + prefix.
 */
export function formatVelocity(velocity: number): string {
	if (velocity >= 1_000) return `+${(velocity / 1_000).toFixed(1)}K`;
	return `+${Math.round(velocity)}`;
}

/**
 * Format a date string as relative time (e.g., "2d ago", "1w ago").
 */
export function formatTimeAgo(dateStr: string | undefined): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
	if (diffDays === 0) return 'today';
	if (diffDays === 1) return '1d ago';
	if (diffDays < 7) return `${diffDays}d ago`;
	if (diffDays < 30) return `${Math.floor(diffDays / 7)}w ago`;
	return `${Math.floor(diffDays / 30)}mo ago`;
}

/**
 * Format a date string as relative time with "Updated" prefix.
 */
export function formatUpdatedAgo(dateStr: string | undefined): string {
	if (!dateStr) return '';
	const date = new Date(dateStr);
	const now = new Date();
	const diffMs = now.getTime() - date.getTime();
	const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
	if (diffDays === 0) return 'Updated today';
	if (diffDays === 1) return 'Updated yesterday';
	if (diffDays < 7) return `Updated ${diffDays}d ago`;
	if (diffDays < 30) return `Updated ${Math.floor(diffDays / 7)}w ago`;
	return `Updated ${Math.floor(diffDays / 30)}mo ago`;
}

/**
 * Truncate text with ellipsis.
 */
export function truncateText(text: string | undefined, maxLen: number): string {
	if (!text) return '';
	if (text.length <= maxLen) return text;
	return text.slice(0, maxLen).trimEnd() + '...';
}
