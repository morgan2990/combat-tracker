// Prompts for confirmation, then DELETEs url. Calls onSuccess if the
// request succeeds; calls onError (if provided) with a message otherwise.
// Does nothing if the user cancels the confirmation.
export async function deleteWithConfirm(
  message: string,
  url: string,
  onSuccess: () => void,
  onError?: (message: string) => void,
) {
  if (!window.confirm(message)) return
  const res = await fetch(url, { method: 'DELETE' })
  if (res.ok) {
    onSuccess()
  } else {
    onError?.('Failed to delete')
  }
}
