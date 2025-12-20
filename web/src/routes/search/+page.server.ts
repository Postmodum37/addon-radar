import { searchAddons } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url }) => {
	const query = url.searchParams.get('q') || '';
	let page = parseInt(url.searchParams.get('page') || '1', 10);
	if (page < 1 || isNaN(page)) page = 1;

	if (!query) {
		return { query, results: null };
	}

	const results = await searchAddons(query, page);

	return {
		query,
		results
	};
};
