import { error } from '@sveltejs/kit';
import { getTrendingHot, getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
	const [hotResult, risingResult] = await Promise.all([
		getTrendingHot(1, 10),
		getTrendingRising(1, 10)
	]);

	// If both API calls failed, show error page
	if (hotResult === null && risingResult === null) {
		console.error('Homepage load failed: both API calls returned null');
		throw error(503, 'Unable to load trending data. Please try again later.');
	}

	return {
		hot: hotResult?.data ?? [],
		rising: risingResult?.data ?? []
	};
};
