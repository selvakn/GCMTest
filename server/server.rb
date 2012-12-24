require 'rubygems'
require 'bundler'
require 'securerandom'
require 'json'
Bundler.require
require  'dm-migrations'

API_KEY = "AIzaSyDN9vs2KvUeDeHqAWT56CqnriFhvZCvL-8"
DataMapper::Logger.new($stdout, :debug)
DataMapper.setup(:default, 'mysql://root@localhost/gcm_test')

class Stats
  include DataMapper::Resource
  property :id, Serial
  property :message_id, Text
  property :post_time, DateTime
  property :received_time, DateTime
  belongs_to :device
end

class Device
  include DataMapper::Resource
  property :id, Serial
  property :registration_id, Text
end

DataMapper.auto_upgrade!

get '/register/:registration_id' do
  Device.create! :registration_id => params[:registration_id]
end

get '/push' do
  devices = Device.all
  gcm = GCM.new(API_KEY)
  message_id = SecureRandom.uuid
  now = Time.now
  options = {
      :data => {
          :id => message_id,
          :sentTimestamp => (now.to_f * 1000.0).to_i
      },
      :collapse_key => "message",
      :time_to_live => 0
  }
  response = gcm.send_notification(devices.map(&:registration_id), options)
  devices.each do |device|
    Stats.create! :device => device, :message_id => message_id, :post_time => now
  end
  [response[:status], response[:headers], response[:body]]
end


get '/received/:registration_id/:message_id' do
  device = Device.first(:registration_id => params[:registration_id])
  stats = Stats.first(:device => device, :message_id => params[:message_id])
  stats.update :received_time => Time.now
end
