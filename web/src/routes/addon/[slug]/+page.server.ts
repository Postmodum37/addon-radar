import { error } from '@sveltejs/kit';
import { getAddon, getAddonHistory } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ params }) => {
	const [addon, history] = await Promise.all([
		getAddon(params.slug),
		getAddonHistory(params.slug)
	]);

	if (!addon) {
		throw error(404, 'Addon not found');
	}

	return {
		addon,
		history
	};
};
