const avatarModules = import.meta.glob<string>("../assets/avatars/*.jpg", { eager: true, query: "?url", import: "default" })

export const avatarURLs = Object.entries(avatarModules)
  .filter(([path]) => !path.endsWith("/default.jpg"))
  .sort(([left], [right]) => left.localeCompare(right, undefined, { numeric: true }))
  .map(([, url]) => url)

export async function avatarDataURL(url: string) {
  const image = new Image()
  image.crossOrigin = "anonymous"
  await new Promise<void>((resolve, reject) => {
    image.onload = () => resolve()
    image.onerror = () => reject(new Error("头像加载失败"))
    image.src = url
  })
  const canvas = document.createElement("canvas")
  canvas.width = 256
  canvas.height = 256
  const context = canvas.getContext("2d")
  if (!context) throw new Error("浏览器不支持头像转换")
  context.drawImage(image, 0, 0, 256, 256)
  return canvas.toDataURL("image/jpeg", 0.88)
}

export async function randomAvatarDataURL() {
  if (!avatarURLs.length) return undefined
  const url = avatarURLs[Math.floor(Math.random() * avatarURLs.length)]
  return avatarDataURL(url)
}
