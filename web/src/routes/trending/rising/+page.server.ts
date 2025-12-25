import { getTrendingRising } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const page = parseInt(url.searchParams.get('page') || '1', 10);
	const result = await getTrendingRising(page, 20);

	return {
		addons: result?.data ?? [],
		meta: result?.meta ?? { page: 1, per_page: 20, total: 0, total_pages: 0 }
	};
};
