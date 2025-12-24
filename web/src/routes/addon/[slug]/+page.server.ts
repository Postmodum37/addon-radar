import { getAddon, getAddonHistory } from '$lib/api';
import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const addon = await getAddon(params.slug);
	if (!addon) {
		throw error(404, 'Addon not found');
	}

	// Get 4 weeks of history (672 hours)
	const history = await getAddonHistory(params.slug, 672);

	// Aggregate by day
	const dailyData = aggregateByDay(history);

	return {
		addon,
		dailyHistory: dailyData
	};
};

interface SnapshotInput {
	recorded_at: string;
	download_count: number;
}

function aggregateByDay(snapshots: SnapshotInput[]) {
	const byDay = new Map<string, { count: number; downloads: number[] }>();

	for (const snap of snapshots) {
		const date = snap.recorded_at.split('T')[0];
		if (!byDay.has(date)) {
			byDay.set(date, { count: 0, downloads: [] });
		}
		const day = byDay.get(date)!;
		day.count++;
		day.downloads.push(snap.download_count);
	}

	// Get last download of each day and calculate delta
	const result: { date: string; download_count: number; downloads_delta: number }[] = [];
	const sortedDates = [...byDay.keys()].sort().reverse();

	for (let i = 0; i < sortedDates.length && i < 28; i++) {
		const date = sortedDates[i];
		const day = byDay.get(date)!;
		const downloads = Math.max(...day.downloads);
		const prevDate = sortedDates[i + 1];
		const prevDownloads = prevDate ? Math.max(...byDay.get(prevDate)!.downloads) : downloads;

		result.push({
			date,
			download_count: downloads,
			downloads_delta: downloads - prevDownloads
		});
	}

	return result.reverse(); // oldest first for chart
}
