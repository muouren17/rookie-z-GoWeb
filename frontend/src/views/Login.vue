<template>
  <div class="main">
    <div class="container">
      <h2 class="form-title">登录</h2>
      <div class="form-group">
        <label for="name">用户名</label>
        <input type="text" class="form-control" v-model.trim="username" maxlength="64" autocomplete="username" name="name" id="name" placeholder="用户名" />
      </div>
      <div class="form-group">
        <label for="pass">密码</label>
        <input type="password" class="form-control" v-model="password" autocomplete="current-password" @keyup.enter="submit"  name="pass" id="pass" placeholder="密码" />
      </div>
      <p v-if="error" class="request-message" role="alert">{{ error }}</p>
      <div class="form-btn">
        <button type="button" class="btn btn-info" :disabled="submitted" @click="submit">{{ submitted ? "登录中…" : "提交" }}</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "Login",
  data() {
    return { username: "", password: "", submitted: false, error: "" };
  },
  methods: {
    async submit() {
      if (this.submitted) return;
      this.error = "";
      if (!this.username.trim() || !this.password) {
        this.error = "请输入用户名和密码";
        return;
      }
      this.submitted = true;
      try {
        const response = await this.$axios({
          method: "post",
          url: "/login",
          data: { username: this.username, password: this.password }
        });
        if (response.code !== 1000) {
          this.error = typeof response.msg === "string" ? response.msg : "登录失败，请检查输入";
          return;
        }
        this.$store.commit("login", response.data);
        const redirect = this.$route.query.redirect;
        const target = typeof redirect === "string" && redirect.startsWith("/") &&
          !redirect.startsWith("//") && redirect !== "/login" ? redirect : "/";
        await this.$router.push({ path: target });
      } catch (error) {
        this.error = error.message || "登录失败，请稍后重试";
      } finally {
        this.submitted = false;
      }
    }
  }
};
</script>
<style lang="less" scoped>
.main {
  background: #f8f8f8;
  padding: 150px 0;
  .container {
    width: 600px;
    background: #fff;
    margin: 0 auto;
    max-width: 1200px;
    padding: 20px;
    .form-title {
      margin-bottom: 33px;
      text-align: center;
    }
    .form-group {
      margin: 15px;
      label {
        display: inline-block;
        max-width: 100%;
        margin-bottom: 5px;
        font-weight: 700;
      }
      .form-control {
        display: block;
        width: 100%;
        height: 34px;
        padding: 6px 12px;
        font-size: 14px;
        line-height: 1.42857143;
        color: #555;
        background-color: #fff;
        background-image: none;
        border: 1px solid #ccc;
        border-radius: 4px;
      }
    }
    .form-btn {
      display: flex;
      justify-content: center;
      .btn {
        padding: 6px 20px;
        font-size: 18px;
        line-height: 1.3333333;
        border-radius: 6px;
        display: inline-block;
        margin-bottom: 0;
        font-weight: 400;
        text-align: center;
        white-space: nowrap;
        vertical-align: middle;
        -ms-touch-action: manipulation;
        touch-action: manipulation;
        cursor: pointer;
        border: 1px solid transparent;
      }
      .btn-info {
        color: #fff;
        background-color: #5bc0de;
        border-color: #46b8da;
      }
    }
  }
}
</style>