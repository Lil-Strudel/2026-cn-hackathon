const time = new Intl.DateTimeFormat(undefined, {
	hour: "numeric",
	minute: "2-digit",
});

const dateTime = new Intl.DateTimeFormat(undefined, {
	weekday: "short",
	hour: "numeric",
	minute: "2-digit",
});

export function formatTime(iso: string) {
	return time.format(new Date(iso));
}

export function formatDateTime(iso: string) {
	return dateTime.format(new Date(iso));
}
