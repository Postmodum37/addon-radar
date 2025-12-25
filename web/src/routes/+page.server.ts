import { getTrendingHot, getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
	const [hotResult, risingResult] = await Promise.all([
		getTrendingHot(1, 10),
		getTrendingRising(1, 10)
	]);

	return {
		hot: hotResult?.data ?? [],
		rising: risingResult?.data ?? []
	};
};
