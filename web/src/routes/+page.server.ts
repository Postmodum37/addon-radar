import { getTrendingHot, getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async () => {
	const [hot, rising] = await Promise.all([
		getTrendingHot(),
		getTrendingRising()
	]);

	return {
		hot,
		rising
	};
};
