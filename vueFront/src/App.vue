<template>
    <div id="app">
        <router-view/>
    </div>
</template>

<script>
    import Home from './Home.vue';
    export default {
    name: "App",
    components: {
        Home,
    },
};
</script>