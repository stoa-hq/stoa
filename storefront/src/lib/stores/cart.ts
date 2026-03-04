import { writable, derived } from 'svelte/store';
import { cartApi, type Cart, type CartItem } from '$lib/api/cart';

const CART_ID_KEY = 'storefront_cart_id';

interface CartState {
	cartId: string | null;
	items: CartItem[];
	loading: boolean;
}

function createCartStore() {
	const initial: CartState = {
		cartId: typeof localStorage !== 'undefined' ? localStorage.getItem(CART_ID_KEY) : null,
		items: [],
		loading: false
	};

	const { subscribe, update, set } = writable<CartState>(initial);

	function saveCartId(id: string) {
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(CART_ID_KEY, id);
		}
	}

	function applyCart(cart: Cart) {
		update((s) => ({ ...s, cartId: cart.id, items: cart.items, loading: false }));
	}

	async function ensureCart(): Promise<string> {
		let cartId: string | null = null;
		const unsubscribe = subscribe((s) => (cartId = s.cartId));
		unsubscribe();

		if (cartId) return cartId;

		update((s) => ({ ...s, loading: true }));
		const res = await cartApi.create();
		const id = res.data!.id;
		saveCartId(id);
		applyCart(res.data!);
		return id;
	}

	return {
		subscribe,

		async load() {
			let cartId: string | null = null;
			const unsubscribe = subscribe((s) => (cartId = s.cartId));
			unsubscribe();
			if (!cartId) return;

			try {
				update((s) => ({ ...s, loading: true }));
				const res = await cartApi.get(cartId);
				if (res.data) applyCart(res.data);
			} catch {
				// Cart expired or not found – reset
				if (typeof localStorage !== 'undefined') localStorage.removeItem(CART_ID_KEY);
				set({ cartId: null, items: [], loading: false });
			}
		},

		async add(productId: string, quantity: number, variantId?: string) {
			const id = await ensureCart();
			update((s) => ({ ...s, loading: true }));
			const res = await cartApi.addItem(id, productId, quantity, variantId);
			if (res.data) applyCart(res.data);
		},

		async updateQuantity(itemId: string, quantity: number) {
			let cartId: string | null = null;
			const unsubscribe = subscribe((s) => (cartId = s.cartId));
			unsubscribe();
			if (!cartId) return;

			update((s) => ({ ...s, loading: true }));
			if (quantity <= 0) {
				const res = await cartApi.removeItem(cartId, itemId);
				if (res.data) applyCart(res.data);
			} else {
				const res = await cartApi.updateItem(cartId, itemId, quantity);
				if (res.data) applyCart(res.data);
			}
		},

		async remove(itemId: string) {
			let cartId: string | null = null;
			const unsubscribe = subscribe((s) => (cartId = s.cartId));
			unsubscribe();
			if (!cartId) return;

			update((s) => ({ ...s, loading: true }));
			const res = await cartApi.removeItem(cartId, itemId);
			if (res.data) applyCart(res.data);
		},

		clear() {
			if (typeof localStorage !== 'undefined') localStorage.removeItem(CART_ID_KEY);
			set({ cartId: null, items: [], loading: false });
		}
	};
}

export const cartStore = createCartStore();
export const cartCount = derived(cartStore, ($cart) =>
	$cart.items.reduce((sum, item) => sum + item.quantity, 0)
);
