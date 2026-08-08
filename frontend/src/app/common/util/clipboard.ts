/**
 * 复制文本到剪贴板。
 * Clipboard API 仅在 HTTPS / localhost 可用；HTTP 局域网访问时回退到 execCommand。
 */
export async function copyToClipboard(text: string): Promise<void> {
	if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText && window.isSecureContext) {
		await navigator.clipboard.writeText(text);
		return;
	}
	const ta = document.createElement('textarea');
	ta.value = text;
	ta.setAttribute('readonly', '');
	ta.style.position = 'fixed';
	ta.style.left = '-9999px';
	ta.style.top = '0';
	document.body.appendChild(ta);
	ta.select();
	ta.setSelectionRange(0, text.length);
	try {
		if (!document.execCommand('copy')) {
			throw new Error('execCommand copy failed');
		}
	} finally {
		document.body.removeChild(ta);
	}
}