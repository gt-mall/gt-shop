module.exports = {
  hooks: {
    onFormatTypeName(typeName, rawTypeName) {
      const names = {
        "user.UserResponse": "UserRow",
        "response.Response-user_UserResponse": "UserApiResponse",
        "response.Response-array_user_UserResponse": "UserListApiResponse",
      };

      return names[rawTypeName] || typeName;
    },
  },
};
