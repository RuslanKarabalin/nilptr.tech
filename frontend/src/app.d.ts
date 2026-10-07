import type { AdminUser } from '$lib/types';

declare global {
	namespace App {
		interface Locals {
			/** Logged in admin, set by hooks.server.ts on /admin routes. */
			user: AdminUser | null;
		}
		interface Error {
			message: string;
		}
	}
}

export {};
