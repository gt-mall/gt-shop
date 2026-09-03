export function getHelloPlugin() {
  return Promise.resolve({
    code: 1,
    data: {
      message: 'Hello Plugin',
      plugin: 'hello',
    },
  });
}
