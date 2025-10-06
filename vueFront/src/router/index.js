import { createRouter, createWebHistory } from 'vue-router';
import Home from '../Home.vue';
import EmployeeDetails from '../EmployeeDetails.vue';
import CreateEmployee from '../CreateEmployee.vue';
import UpdateEmployee from '../UpdateEmployee.vue';

const routes = [
  { path: '/', component: Home },
  { path: '/list', component: EmployeeDetails },
  { path: '/create', component: CreateEmployee },
  { path: '/update/:id', component: UpdateEmployee }, 
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

export default router;
