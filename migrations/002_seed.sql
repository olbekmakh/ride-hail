begin;

-- PASSENGER
insert into users (id,email,role,password_hash)
values ('550e8400-e29b-41d4-a716-446655440001','p@demo.com','PASSENGER','x')
on conflict do nothing;

-- DRIVER
insert into users (id,email,role,password_hash)
values ('660e8400-e29b-41d4-a716-446655440001','d@demo.com','DRIVER','x')
on conflict do nothing;

insert into drivers (id,license_number,vehicle_type,status)
values ('660e8400-e29b-41d4-a716-446655440001','LIC123','ECONOMY','OFFLINE')
on conflict do nothing;

-- ADMIN
insert into users (id,email,role,password_hash)
values ('770e8400-e29b-41d4-a716-446655440001','a@demo.com','ADMIN','x')
on conflict do nothing;

commit;
