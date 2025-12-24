import { getTrendingHot } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const perPage = 20;

	const allHot = await getTrendingHot();
	const totalPages = Math.ceil(allHot.length / perPage);
	const start = (page - 1) * perPage;
	const addons = allHot.slice(start, start + perPage);

	return {
		addons,
		meta: {
			page,
			perPage,
			total: allHot.length,
			totalPages
		}
	};
};
