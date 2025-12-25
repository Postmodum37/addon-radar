import { error } from '@sveltejs/kit';
import { getTrendingHot } from '$lib/api';
import type { PageServerLoad } from './$types';

function parsePageParam(value: string | null): number {
	if (!value) return 1;
	const parsed = parseInt(value, 10);
	return isNaN(parsed) || parsed < 1 ? 1 : parsed;
}

export const load: PageServerLoad = async ({ url }) => {
	const page = parsePageParam(url.searchParams.get('page'));
	const result = await getTrendingHot(page, 20);

	if (result === null) {
		console.error('Trending hot page load failed', { page });
		throw error(503, 'Unable to load trending data. Please try again later.');
	}

	return {
		addons: result.data,
		meta: result.meta
	};
};
