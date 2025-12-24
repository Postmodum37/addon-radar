import { getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const perPage = 20;

	const allRising = await getTrendingRising();
	const totalPages = Math.ceil(allRising.length / perPage);
	const start = (page - 1) * perPage;
	const addons = allRising.slice(start, start + perPage);

	return {
		addons,
		meta: {
			page,
			perPage,
			total: allRising.length,
			totalPages
		}
	};
};
